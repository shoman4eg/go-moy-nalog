package moynalog

import (
	"context"
	"net/http"
	"testing"
)

func TestUsersGet(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{
			"lastName": null,
			"id": 3147396,
			"displayName": "ПУПКИН ВАСИЛИЙ ВАСИЛЬЕВИЧ",
			"middleName": null,
			"email": "user@example.com",
			"phone": "79000000000",
			"inn": "770000000000",
			"snils": "114-880-270 52",
			"avatarExists": false,
			"initialRegistrationDate": "2021-01-27T22:38:30.057957Z",
			"registrationDate": "2021-01-27T22:38:30.057957Z",
			"firstReceiptRegisterTime": "2021-03-11T13:37:23Z",
			"firstReceiptCancelTime": null,
			"hideCancelledReceipt": false,
			"registerAvailable": null,
			"status": "ACTIVE",
			"restrictedMode": false,
			"pfrUrl": null,
			"login": null
		}`)
	})

	user, _, err := client.Users.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if user.ID != 3147396 {
		t.Errorf("ID = %d, want 3147396", user.ID)
	}
	if user.Inn != "770000000000" {
		t.Errorf("Inn = %q, want %q", user.Inn, "770000000000")
	}
	if user.Status != "ACTIVE" {
		t.Errorf("Status = %q, want %q", user.Status, "ACTIVE")
	}
	if user.RegistrationDate.IsZero() {
		t.Error("RegistrationDate must be decoded")
	}
	// Nulls must decode to zero values rather than failing.
	if !user.FirstReceiptCancelTime.IsZero() {
		t.Error("a null firstReceiptCancelTime must decode to the zero time")
	}
	if user.LastName != "" {
		t.Errorf("LastName = %q, want it empty for null", user.LastName)
	}
}

// The nullable profile fields must also decode when the API does fill them in.
func TestUsersGetPopulatedOptionalFields(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/user", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{
			"id": 1000000,
			"displayName": "Surname Name MiddleName",
			"lastName": "Surname",
			"middleName": "MiddleName",
			"email": "email@example.com",
			"phone": "79000000000",
			"inn": "300000000000",
			"snils": "000-000-000 00",
			"avatarExists": false,
			"hideCancelledReceipt": false,
			"registerAvailable": null,
			"status": "ACTIVE",
			"restrictedMode": false,
			"pfrUrl": "https://es.pfrf.ru",
			"login": "100000000000"
		}`)
	})

	user, _, err := client.Users.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if user.LastName != "Surname" {
		t.Errorf("LastName = %q, want %q", user.LastName, "Surname")
	}
	if user.MiddleName != "MiddleName" {
		t.Errorf("MiddleName = %q, want %q", user.MiddleName, "MiddleName")
	}
	if user.Snils != "000-000-000 00" {
		t.Errorf("Snils = %q, want the SNILS", user.Snils)
	}
	if user.PfrURL != "https://es.pfrf.ru" {
		t.Errorf("PfrURL = %q, want %q", user.PfrURL, "https://es.pfrf.ru")
	}
	if user.Login != "100000000000" {
		t.Errorf("Login = %q, want %q", user.Login, "100000000000")
	}
	if user.RegisterAvailable != nil {
		t.Errorf("RegisterAvailable = %v, want nil", user.RegisterAvailable)
	}
}
