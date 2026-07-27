# Неофициальный Go API клиент [lknpd.nalog.ru](https://lknpd.nalog.ru/) ("Мой Налог")

[![Go Reference](https://pkg.go.dev/badge/github.com/shoman4eg/go-moy-nalog.svg)](https://pkg.go.dev/github.com/shoman4eg/go-moy-nalog/moynalog)
[![Go version](https://img.shields.io/github/go-mod/go-version/shoman4eg/go-moy-nalog?style=flat-square)](go.mod)
[![Latest Version](https://img.shields.io/github/release/shoman4eg/go-moy-nalog.svg?style=flat-square)](https://github.com/shoman4eg/go-moy-nalog/releases)
[![Tests](https://img.shields.io/github/actions/workflow/status/shoman4eg/go-moy-nalog/ci.yml?branch=master&style=flat-square&label=tests)](https://github.com/shoman4eg/go-moy-nalog/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/shoman4eg/go-moy-nalog?style=flat-square)](LICENSE)
[![Donate](https://img.shields.io/badge/Donate-Cloudtips-6496dc?style=flat-square)](https://pay.cloudtips.ru/p/2e70e850)

Позволяет автоматизировать отправку информации о доходах для самозанятых,
получать информацию о созданных чеках и удалять их. Поддерживается
аутентификация по ИНН и паролю, а также по номеру телефона.

Go-порт PHP-библиотеки [shoman4eg/moy-nalog](https://github.com/shoman4eg/moy-nalog).
Структура клиента повторяет [google/go-github](https://github.com/google/go-github).

## Установка

```bash
go get github.com/shoman4eg/go-moy-nalog
```

Требуется Go 1.25 или новее.

```go
import "github.com/shoman4eg/go-moy-nalog/moynalog"
```

## Использование

### Создание клиента

```go
// Клиент с настройками по умолчанию
client := moynalog.NewClient()

// Свой *http.Client — например, с прокси и таймаутом
proxyURL, _ := url.Parse("http://12.34.56.78:3128")
httpClient := &http.Client{
    Timeout:   30 * time.Second,
    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
}
client := moynalog.NewClient(moynalog.WithHTTPClient(httpClient))

// Другой endpoint или версия API (по умолчанию https://lknpd.nalog.ru/api и v1)
client := moynalog.NewClient(
    moynalog.WithEndpoint("https://lknpd.nalog.ru/api"),
    moynalog.WithVersion("v1"),
    moynalog.WithUserAgent("my-app/1.0"),
)
```

Все таймауты и ретраи транспортного уровня задаются через ваш `*http.Client` —
библиотека своего таймаута не навязывает. Каждый метод принимает
`context.Context`, отмена контекста прерывает запрос.

### Device ID

`deviceId` вычисляется один раз при создании клиента и дальше не меняется:
API привязывает чеки и refresh-токен к устройству, с которого они созданы.

```go
// По умолчанию — PlatformIDStrategy, deviceId стабилен для одной машины
client := moynalog.NewClient()

// Зафиксировать конкретный deviceId
client := moynalog.NewClient(moynalog.WithDeviceID("abcdefghijklmnopqrstu"))

// Полезно, если вы обслуживаете несколько самозанятых:
// свой стабильный deviceId на каждого пользователя
client := moynalog.NewClient(
    moynalog.WithDeviceIDGenerator(moynalog.NewStaticDeviceIDGenerator("example id")),
)

// Случайный deviceId на каждый запуск
client := moynalog.NewClient(
    moynalog.WithDeviceIDGenerator(moynalog.NewRandomDeviceIDGenerator()),
)

// Своя стратегия
client := moynalog.NewClient(
    moynalog.WithDeviceIDGenerator(moynalog.NewDeviceIDGenerator(
        moynalog.IDStrategyFunc(func() (string, error) {
            return "my raw material", nil
        }),
        moynalog.WithIDLength(21),
        moynalog.WithIDLowercase(true),
    )),
)

// Или полностью свой генератор
client := moynalog.NewClient(
    moynalog.WithDeviceIDGenerator(moynalog.DeviceIDFunc(func() (string, error) {
        return loadDeviceIDFromDB()
    })),
)
```

### Логирование запросов (опционально)

Логирование включается своим `http.RoundTripper` — библиотека не тянет
логгер в зависимости.

```go
type loggingTransport struct {
    base   http.RoundTripper
    logger *slog.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    resp, err := t.base.RoundTrip(req)
    if err != nil {
        t.logger.Error("request failed", "method", req.Method, "url", req.URL, "err", err)

        return nil, err
    }
    t.logger.Info("request", "method", req.Method, "url", req.URL, "status", resp.StatusCode)

    return resp, nil
}

client := moynalog.NewClient(moynalog.WithHTTPClient(&http.Client{
    Transport: &loggingTransport{base: http.DefaultTransport, logger: logger},
}))
```

Не логируйте заголовок `Authorization` и тело запросов аутентификации — там
токены и пароль.

### Аутентификация

При аутентификации методами `CreateAccessToken` (по ИНН и паролю) или
`CreateAccessTokenByPhone` (по номеру телефона) вместе с токеном доступа
возвращается токен обновления (**refreshToken**) с неограниченным сроком
действия. Сохраните весь `*AccessToken` целиком и переиспользуйте его через
`WithToken`.

> При повторном вызове `CreateAccessToken` и `CreateAccessTokenByPhone`
> предыдущий accessToken становится недействительным.

Клиент сам обновляет протухший токен: при ответе 401 он вызывает
`Auth.Refresh` и повторяет запрос (не более 2 раз). Актуальный токен можно
забрать методом `Token()` и сохранить.

#### С помощью ИНН и пароля

> Если Вам нужно восстановить пароль от сервиса ["Мой налог"](https://lknpd.nalog.ru/),
> это возможно сделать только через
> ["Личный кабинет налогоплательщика"](https://lkfl2.nalog.ru/lkfl/login).
> Аккаунты на обоих сервисах одинаковые.

```go
client := moynalog.NewClient()

token, _, err := client.Auth.CreateAccessToken(ctx, username, password)
if err != nil {
    if errors.Is(err, moynalog.ErrUnauthorized) {
        log.Print("неверные логин или пароль")
    }

    return err
}

// Аутентифицированный клиент
client = client.WithToken(token)
```

Сохранить и восстановить токен между запусками:

```go
raw, err := json.Marshal(client.Token())
// ...
token := new(moynalog.AccessToken)
if err := json.Unmarshal(raw, token); err != nil {
    return err
}
client := moynalog.NewClient().WithToken(token)
```

#### По номеру телефона

Аутентификация по номеру телефона происходит в 2 шага:

1. Запросите SMS с кодом подтверждения и сохраните возвращённый **challengeToken**;
2. Обменяйте номер телефона, **challengeToken** и код подтверждения на **accessToken**.

> **Внимание:** запрос нового кода подтверждения возможен только если предыдущий
> код истёк (2 минуты), или по предыдущему коду произошла успешная
> аутентификация. Повторная отправка выпущенного кода невозможна, только
> одновременно с созданием нового.

```go
// Шаг 1. Запросить SMS
client := moynalog.NewClient()

challenge, _, err := client.Auth.CreatePhoneChallenge(ctx, "79000000000")
if err != nil {
    return err
}
// challenge.ChallengeToken — "00000000-0000-0000-0000-000000000000"
// challenge.ExpireDate     — moynalog.Time
// challenge.ExpireIn       — 120

// Шаг 2. Обменять код из SMS на токен
token, _, err := client.Auth.CreateAccessTokenByPhone(
    ctx,
    "79000000000",                          // Номер телефона
    challenge.ChallengeToken,               // challengeToken из шага 1
    "123456",                               // Код из СМС
)
if err != nil {
    return err
}

client = client.WithToken(token)
```

### Создать чек c контрагентом по умолчанию (физ. лицо)

```go
created, _, err := client.Income.CreateItem(
    ctx,
    "Предоставление информационных услуг #970/2495", // Наименование
    decimal.NewFromFloat(1800.30),                   // Стоимость в рублях
    decimal.NewFromInt(1),                           // Количество
)
if err != nil {
    return err
}

// UUID чека для операций запроса данных чека или его отмены
receiptUUID := created.ApprovedReceiptUUID
```

### Создать чек с несколькими позициями

```go
created, _, err := client.Income.Create(ctx, &moynalog.IncomeCreateRequest{
    Services: []moynalog.IncomeServiceItem{
        {
            Name:     "Предоставление информационных услуг #970/2495",
            Amount:   decimal.NewFromFloat(1800.30),
            Quantity: decimal.NewFromInt(1),
        },
        {
            Name:     "Предоставление информационных услуг #971/2495",
            Amount:   decimal.NewFromInt(900),
            Quantity: decimal.NewFromInt(2),
        },
    },
    // Дата продажи, по умолчанию — сейчас
    OperationTime: time.Date(2020, time.December, 31, 12, 12, 0, 0, time.Local),
})
```

`TotalAmount` считает сама библиотека как сумму `Amount × Quantity` по всем
позициям. Суммы — `decimal.Decimal` из
[shopspring/decimal](https://github.com/shopspring/decimal), чтобы не терять
копейки на float.

Часовой пояс берётся из переданного `time.Time`: используйте
`time.Local`, `time.LoadLocation("Europe/Kaliningrad")` или нужную зону явно.

### Создать чек для указанного типа контрагента

```go
// По умолчанию физ. лицо без указания контактных данных
counterparty := &moynalog.IncomeClient{}

// Или физ. лицо с указанием контактных данных
counterparty := &moynalog.IncomeClient{
    ContactPhone: "+79009000000",
    DisplayName:  "Вася Пупкин",
    IncomeType:   moynalog.IncomeTypeIndividual,
    Inn:          "390000000000", // ИНН физ. лица (12 символов)
}

// Или юр. лицо (ИП, ООО и т.п.)
counterparty := &moynalog.IncomeClient{
    DisplayName: "ИП Вася Пупкин Валерьевич",
    IncomeType:  moynalog.IncomeTypeLegalEntity,
    Inn:         "7700000000", // ИНН юр. лица (10 символов)
}

// Или иностранная организация
counterparty := &moynalog.IncomeClient{
    DisplayName: "Facebook Inc.",
    IncomeType:  moynalog.IncomeTypeForeignAgency,
    Inn:         "9909000000",
}

created, _, err := client.Income.Create(ctx, &moynalog.IncomeCreateRequest{
    Services: []moynalog.IncomeServiceItem{{
        Name:     "Предоставление информационных услуг #970/2495",
        Amount:   decimal.NewFromFloat(1800.30),
        Quantity: decimal.NewFromInt(1),
    }},
    Client: counterparty,
})
```

### Создать счёт на оплату (invoice)

Счёт на оплату выставляется для оплаты по банковским реквизитам
(тип оплаты `ACCOUNT`).

```go
invoice, _, err := client.Invoice.CreateItem(
    ctx,
    "Предоставление информационных услуг #970/2495",
    decimal.NewFromFloat(1800.30),
    decimal.NewFromInt(1),
)

// Или с несколькими позициями и своей датой
invoice, _, err := client.Invoice.Create(ctx, &moynalog.InvoiceCreateRequest{
    Services: []moynalog.InvoiceServiceItem{{
        Name:     "Предоставление информационных услуг #970/2495",
        Amount:   decimal.NewFromFloat(1800.30),
        Quantity: decimal.NewFromInt(1),
    }},
    OperationTime: operationTime,
})
```

> Методы `Invoice.Cancel()` и `Invoice.UpdatePaymentInfo()` не реализованы
> в API и возвращают `moynalog.ErrNotImplemented`.

### Получить список чеков

```go
// Получить список чеков (по умолчанию 100 последних)
incomes, _, err := client.Income.List(ctx, nil)

// Фильтрация
incomes, _, err := client.Income.List(ctx, &moynalog.IncomeListOptions{
    From:        moynalog.NewTime(from),             // начало периода (по умолчанию: нет)
    To:          moynalog.NewTime(to),               // конец периода (по умолчанию: нет)
    Offset:      10,                                 // смещение для пагинации (по умолчанию: 0)
    Limit:       25,                                 // количество результатов (по умолчанию: 100, максимум: 100)
    BuyerType:   moynalog.BuyerTypePerson,           // тип покупателя (по умолчанию: нет)
    ReceiptType: moynalog.ReceiptTypeRegistered,     // тип чека (по умолчанию: нет)
    SortBy:      moynalog.SortByOperationTimeAsc,    // сортировка (по умолчанию: SortByOperationTimeDesc)
})

for _, item := range incomes.Content {
    fmt.Println(item.ApprovedReceiptUUID, item.TotalAmount, item.Cancelled())
}
if incomes.HasMore {
    // есть следующая страница
}
```

Возможные значения фильтрации:

| Константа                          | Значение                    |
|------------------------------------|-----------------------------|
| `BuyerTypePerson`                  | физлицо                     |
| `BuyerTypeCompany`                 | юрлицо                      |
| `BuyerTypeForeignAgency`           | иностранная организация     |
| `ReceiptTypeRegistered`            | действителен                |
| `ReceiptTypeCancelled`             | аннулирован                 |
| `SortByOperationTimeAsc`           | дата: сначала старые        |
| `SortByOperationTimeDesc`          | дата: сначала новые         |
| `SortByTotalAmountAsc`             | стоимость: по возрастанию   |
| `SortByTotalAmountDesc`            | стоимость: по убыванию      |

### Получить чек (скан-копия) или данные чека в JSON формате

```go
receiptUUID := "20hykdxbp8"

// Ссылка на чек для печати
printURL, err := client.Receipt.PrintURL(ctx, receiptUUID)

// Скачать печатную форму чека (PDF)
pdf, _, err := client.Receipt.Print(ctx, receiptUUID)

// Данные по чеку
receipt, _, err := client.Receipt.JSON(ctx, receiptUUID)
```

ИНН для этих запросов берётся из профиля в токене; если профиля нет, клиент
сам сходит за ним в `/user`.

### Отменить чек

```go
receiptUUID := "20hykdxbp8"

cancelled, _, err := client.Income.Cancel(ctx, &moynalog.IncomeCancelRequest{
    ReceiptUUID: receiptUUID,
    // Причина отмены: "Чек сформирован ошибочно"
    Comment: moynalog.CancelCommentMistake,
    // Или причина отмены: "Возврат средств"
    // Comment: moynalog.CancelCommentRefund,

    OperationTime: operationTime, // дата возврата (по умолчанию: сейчас)
    RequestTime:   requestTime,   // дата запроса отмены (по умолчанию: сейчас)
    PartnerCode:   "",            // код партнёра (по умолчанию: нет)
})
```

### Получить информацию о текущем пользователе

```go
user, _, err := client.Users.Get(ctx)
```

### Получить информацию о необходимых платежах

```go
tax, _, err := client.Tax.Get(ctx)
```

### Получить информацию о платежах

```go
// Второй аргумент — OKTMO ("" для всех регионов), третий — только оплаченные
payments, _, err := client.Tax.Payments(ctx, "", false)
```

### Получить информацию о прошлых платежах

```go
history, _, err := client.Tax.History(ctx, "")
```

### Задолженность и налоговый бонус (taxpayer)

```go
// Информация о задолженности
debts, _, err := client.Taxpayer.Debts(ctx)
debts.HasDebts    // bool — есть ли задолженность
debts.TotalUnpaid // decimal.Decimal — сумма неоплаченного
debts.Debts       // decimal.Decimal

// Налоговый бонус (вычет) и лимиты годового дохода
bonus, _, err := client.Taxpayer.Bonus(ctx)
bonus.BonusAmount                      // остаток бонуса
bonus.TotalIncomeAmount                // доход за текущий год
bonus.AnnualIncomeThreshold            // годовой лимит (2 400 000)
bonus.AvailableIncomeToExceedThreshold // сколько осталось до лимита
bonus.AnnualIncomeStatus               // string — например, "NORMAL"

// Разбивка дохода по годам: map[string]*AnnualIncome
for year, annual := range bonus.TotalIncomeByYears {
    fmt.Println(year, annual.TotalIncomeAmount)
}
```

### Способы оплаты (банковские карты / счета)

```go
// Список сохранённых способов оплаты
accounts, _, err := client.PaymentType.Table(ctx)

// Избранный (по умолчанию) способ оплаты, либо nil
favorite, _, err := client.PaymentType.Favorite(ctx)
```

### Обработка ошибок

Ошибки API оборачиваются в `*moynalog.ErrorResponse` и разворачиваются в
одну из sentinel-ошибок, так что работают оба способа:

```go
user, resp, err := client.Users.Get(ctx)

switch {
case errors.Is(err, moynalog.ErrUnauthorized): // 401
case errors.Is(err, moynalog.ErrValidation):   // 400
case errors.Is(err, moynalog.ErrForbidden):    // 403
case errors.Is(err, moynalog.ErrNotFound):     // 404
case errors.Is(err, moynalog.ErrClient):       // 406
case errors.Is(err, moynalog.ErrPhone):        // 422
case errors.Is(err, moynalog.ErrServer):       // 500
case errors.Is(err, moynalog.ErrUnknown):      // прочее
}

var errResp *moynalog.ErrorResponse
if errors.As(err, &errResp) {
    fmt.Println(errResp.StatusCode(), errResp.Code, errResp.Message, errResp.AdditionalInfo)
}

// resp — обёртка над *http.Response, возвращается и вместе с ошибкой
```

## Известные проблемы

### Проблема [#47](https://github.com/shoman4eg/moy-nalog/issues/47): Не приходят СМС для получения токена

- **Описание**: Проблемы с получением СМС-кода для авторизации
- **Решение**: Проблема связана с API сервиса nalog.ru, временная недоступность сервиса отправки СМС

### Проблема [#22](https://github.com/shoman4eg/moy-nalog/issues/22): Авторизация по телефону или внешним ключам

- **Описание**: Невозможность авторизоваться без пароля (только по номеру телефона)
- **Решение**: Для получения пароля нужно восстановить его через веб-интерфейс "Мой налог"

### Проблема [#49](https://github.com/shoman4eg/moy-nalog/issues/49): Токен истекает несмотря на наличие RefreshToken

- **Описание**: RefreshToken не срабатывает для получения нового токена
- **Настоящая причина**: Токен API "Мой налог" привязывается к конкретному IP-адресу или Device ID, с которого он был сгенерирован
- **Решение**: Токен нужно генерировать на нужном окружении. Токен, созданный с одного IP-адреса/Device ID, не будет работать с другого. Можно попробовать использовать в обоих окружениях `NewStaticDeviceIDGenerator`, чтобы удостовериться, что refresh-токен привязывается к IP, с которого сделан запрос
- **Рекомендация**: Генерировать токен на том же сервере, с которого будут выполняться API-запросы

### Проблема [#21](https://github.com/shoman4eg/moy-nalog/issues/21): Ошибка Could not resolve host: lknpd.nalog.ru

- **Описание**: Невозможность резолва DNS для API сервера
- **Решение**: Временная проблема на стороне сервера или DNS. Проверить настройки DNS на сервере

### Проблема: Ошибка проверки refresh токена

- **Описание**: Одна из причин — на одном сервере обслуживается множество пользователей и Device ID одинаковый для всех. Можно получить такой ответ:

```json
{
  "code": "authentication.failed",
  "message": "Устройство <GENERATED DEVICE ID> для пользователя <INN> не может быть зарегистрировано/обновлено. Так как устройство заблокировано. Причина: Согласно обращению УОК о неправомерном доступе к ЛК",
  "additionalInfo": {}
}
```

- **Решение**: Использовать отдельный Device ID на каждого пользователя:

```go
client := moynalog.NewClient(
    moynalog.WithDeviceIDGenerator(moynalog.NewStaticDeviceIDGenerator("<example id>")),
)
```

## Разработка

```bash
make init    # установить golangci-lint в ./bin
make format  # gofumpt + goimports + gci
make lint    # golangci-lint + go-consistent
make test    # go test -race -cover ./...
make check   # всё сразу, без изменения файлов — то же, что гоняет CI
make help    # список всех целей
```

Подробнее — в [CONTRIBUTING.md](CONTRIBUTING.md).

## Использованные ресурсы

PHP-реализация: [shoman4eg/moy-nalog](https://github.com/shoman4eg/moy-nalog)

Статья на Habr: [Автоматизация для самозанятых: как интегрировать налог с IT проектом](https://habr.com/ru/post/436656/)

Реализация на JS: [alexstep/moy-nalog](https://github.com/alexstep/moy-nalog)

## На кофе

Если этот проект поможет Вам сократить время разработки, вы можете угостить
меня чашкой кофе :)

[Сделать пожертвование автору](https://pay.cloudtips.ru/p/2e70e850)

## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
