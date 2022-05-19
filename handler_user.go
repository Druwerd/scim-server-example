// Original code from: https://github.com/scim2/example-server/blob/master/handler_user.go

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/google/uuid"
	"github.com/scim2/example-server/mdb"
	"github.com/scim2/tools/marshal"
)

var userType = scim.ResourceType{
	ID:          optional.NewString("User"),
	Name:        "User",
	Endpoint:    "/Users",
	Description: optional.NewString("User Account"),
	Schema:      schema.CoreUserSchema(),
	SchemaExtensions: []scim.SchemaExtension{
		{Schema: schema.ExtensionEnterpriseUser()},
	},
	Handler: newUsersResourceHandler(),
}

type usersResourceHandler struct {
	db *mdb.DB
}

func newUsersResourceHandler() *usersResourceHandler {
	return &usersResourceHandler{
		db: mdb.New(),
	}
}

func (u usersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	// Convert attributes to a User struct.
	user, err := attr2User(attributes)
	if err != nil {
		// The SCIM server will convert this error to a InternalServerError (500).
		return scim.Resource{}, err
	}

	// Create a unique identifier.
	user.ID = uuid.New().String()

	// Meta data.
	now := time.Now()
	meta := scim.Meta{
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("v%s", user.ID),
	}

	// Write to database.
	if err := u.db.Update(func(tx *mdb.TX) error {
		tx.Set(user.ID, mdb.Instance{
			Value: user,
			Meta:  meta,
		})
		return nil
	}); err != nil {
		return scim.Resource{}, err
	}
	return user2Resource(user, meta)
}

func (u usersResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	var user mdb.Instance
	// Get user from database.
	if err := u.db.View(func(tx *mdb.TX) error {
		var ok bool
		user, ok = tx.Get(id)
		if !ok {
			// Return NotFound (404) if not found.
			return errors.ScimErrorResourceNotFound(id)
		}
		return nil
	}); err != nil {
		return scim.Resource{}, err
	}
	return user2Resource(user.Value.(User), user.Meta)
}

func (u usersResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	var (
		resources = make([]scim.Resource, 0)
		total     int
	)
	if err := u.db.View(func(tx *mdb.TX) error {
		var (
			i   = 1
			all = tx.GetAll()
		)
		total = len(all)
		for _, v := range all {
			if i > (params.StartIndex + params.Count - 1) {
				break
			}
			if i >= params.StartIndex {
				resource, err := user2Resource(v.Value.(User), v.Meta)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
			i++
		}
		return nil
	}); err != nil {
		return scim.Page{}, err
	}
	return scim.Page{
		TotalResults: total,
		Resources:    resources,
	}, nil
}

func (u usersResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	user, err := attr2User(attributes)
	if err != nil {
		return scim.Resource{}, err
	}
	var meta scim.Meta
	if err := u.db.Update(func(tx *mdb.TX) error {
		previous, ok := tx.Get(id)
		if !ok {
			// Return NotFound (404) if not found.
			return errors.ScimErrorResourceNotFound(id)
		}

		// Take over the identifier.
		user.ID = previous.Value.(User).ID

		// Update meta.
		now := time.Now()
		meta = scim.Meta{
			Created:      previous.Meta.Created,
			LastModified: &now,
			Version:      previous.Meta.Version,
		}

		// Replace instance.
		tx.Set(id, mdb.Instance{
			Value: user,
			Meta:  meta,
		})
		return nil
	}); err != nil {
		return scim.Resource{}, err
	}
	return user2Resource(user, meta)
}

func (u usersResourceHandler) Delete(r *http.Request, id string) error {
	return u.db.Update(func(tx *mdb.TX) error {
		if ok := tx.Delete(id); !ok {
			// Return NotFound (404) if not found.
			return errors.ScimErrorResourceNotFound(id)
		}
		return nil
	})
}

// func (u usersResourceHandler) Patch(r *http.Request, id string, request scim.PatchRequest) (scim.Resource, error) {
func (h usersResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	return scim.Resource{}, &errors.ScimError{
		Status: http.StatusNotImplemented,
	}
}

func attr2User(attributes scim.ResourceAttributes) (User, error) {
	var user User
	if err := marshal.Unmarshal(attributes, &user); err != nil {
		return User{}, err
	}
	return user, nil
}

func user2Resource(user User, meta scim.Meta) (scim.Resource, error) {
	attributes, err := user2Attr(user)
	if err != nil {
		return scim.Resource{}, err
	}
	eID := optional.String{}
	if user.ExternalID != "" {
		eID = optional.NewString(user.ExternalID)
	}
	return scim.Resource{
		ID:         user.ID,
		ExternalID: eID,
		Attributes: attributes,
		Meta:       meta,
	}, nil
}

func user2Attr(user User) (scim.ResourceAttributes, error) {
	return marshal.Marshal(user)
}
