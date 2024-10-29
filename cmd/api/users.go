package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/jandiralceu/greenlight/internal/data"
	"github.com/jandiralceu/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	if err := user.Password.Set(input.Password); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Users.Insert(user); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	if err := app.models.Permissions.AddForUser(user.ID, "movies:read"); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		values := map[string]interface{}{
			"activationToken": token.PlainText,
			"userID":          user.ID,
		}

		if err := app.mailer.Send(user.Email, "user_welcome.tmpl", values); err != nil {
			app.logger.Error(err.Error())
		}
	})

	if err := app.writeJSON(w, http.StatusCreated, user, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
