// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package applications

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/microsoft-graph/applications/stable/application"
	"github.com/hashicorp/go-azure-sdk/microsoft-graph/common-types/stable"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-provider-azuread/internal/clients"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers/consistency"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers/credentials"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers/tf"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azuread/internal/helpers/tf/validation"
	"github.com/hashicorp/terraform-provider-azuread/internal/services/applications/migrations"
	"github.com/hashicorp/terraform-provider-azuread/internal/services/applications/parse"
)

func applicationPasswordResource() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		CreateContext: applicationPasswordResourceCreate,
		ReadContext:   applicationPasswordResourceRead,
		DeleteContext: applicationPasswordResourceDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(15 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(5 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(5 * time.Minute),
		},

		SchemaVersion: 1,
		StateUpgraders: []pluginsdk.StateUpgrader{
			{
				Type:    migrations.ResourceApplicationPasswordInstanceResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: migrations.ResourceApplicationPasswordInstanceStateUpgradeV0,
				Version: 0,
			},
		},

		Schema: map[string]*pluginsdk.Schema{
			"application_id": {
				Description:  "The resource ID of the application for which this password should be created",
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true, // TODO remove Computed in v3.0
				ForceNew:     true,
				ExactlyOneOf: []string{"application_id", "application_object_id"},
				ValidateFunc: parse.ValidateApplicationID,
			},

			"application_object_id": {
				Description:  "The object ID of the application for which this password should be created",
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"application_id", "application_object_id"},
				Deprecated:   "The `application_object_id` property has been replaced with the `application_id` property and will be removed in version 3.0 of the AzureAD provider",
				ValidateFunc: validation.Any(validation.IsUUID, parse.ValidateApplicationID),
				DiffSuppressFunc: func(_, oldValue, newValue string, _ *pluginsdk.ResourceData) bool {
					// Where oldValue is a UUID (i.e. the bare object ID), and newValue is a properly formed application
					// resource ID, we'll ignore a diff where these point to the same application resource.
					// This maintains compatibility with configurations mixing the ID attributes, e.g.
					//     application_object_id = azuread_application.example.id
					if _, err := uuid.ParseUUID(oldValue); err == nil {
						if applicationId, err := parse.ParseApplicationID(newValue); err == nil {
							if applicationId.ApplicationId == oldValue {
								return true
							}
						}
					}
					return false
				},
			},

			"display_name": {
				Description: "A display name for the password",
				Type:        pluginsdk.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},

			"start_date": {
				Description:  "The start date from which the password is valid, formatted as an RFC3339 date string (e.g. `2018-01-01T01:02:03Z`). If this isn't specified, the current date is used",
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},

			"end_date": {
				Description:   "The end date until which the password is valid, formatted as an RFC3339 date string (e.g. `2018-01-01T01:02:03Z`)",
				Type:          pluginsdk.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"end_date_relative"},
				ValidateFunc:  validation.IsRFC3339Time,
			},

			"end_date_relative": {
				Description:   "A relative duration for which the password is valid until, for example `240h` (10 days) or `2400h30m`. Changing this field forces a new resource to be created",
				Type:          pluginsdk.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"end_date"},
				ValidateFunc:  validation.StringIsNotEmpty,
			},

			"rotate_when_changed": {
				Description: "Arbitrary map of values that, when changed, will trigger rotation of the password",
				Type:        pluginsdk.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},

			"key_id": {
				Description: "A UUID used to uniquely identify this password credential",
				Type:        pluginsdk.TypeString,
				Computed:    true,
			},

			"value": {
				Description: "The password for this application, which is generated by Azure Active Directory",
				Type:        pluginsdk.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func applicationPasswordResourceCreate(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics { //nolint
	client := meta.(*clients.Client).Applications.ApplicationClient

	var applicationId *stable.ApplicationId
	var err error
	if v := d.Get("application_id").(string); v != "" {
		if applicationId, err = stable.ParseApplicationID(v); err != nil {
			return tf.ErrorDiagPathF(err, "application_id", "Parsing `application_id`: %q", v)
		}
	} else {
		// TODO: this permits parsing the application_object_id as either a structured ID or a bare UUID, to avoid
		// breaking users who might have `application_object_id = azuread_application.foo.id` in their config, and
		// should be removed in version 3.0 along with the application_object_id property
		v = d.Get("application_object_id").(string)
		if _, err = uuid.ParseUUID(v); err == nil {
			applicationId = pointer.To(stable.NewApplicationID(v))
		} else {
			if applicationId, err = stable.ParseApplicationID(v); err != nil {
				return tf.ErrorDiagPathF(err, "application_id", "Parsing `application_object_id`: %q", v)
			}
		}
	}

	credential, err := credentials.PasswordCredentialForResource(d)
	if err != nil {
		attr := ""
		if kerr, ok := err.(credentials.CredentialError); ok {
			attr = kerr.Attr()
		}
		return tf.ErrorDiagPathF(err, attr, "Generating password credentials for %s", applicationId)
	}
	if credential == nil {
		return tf.ErrorDiagF(errors.New("nil credential was returned"), "Generating password credentials for %s", applicationId)
	}

	tf.LockByName(applicationResourceName, applicationId.ApplicationId)
	defer tf.UnlockByName(applicationResourceName, applicationId.ApplicationId)

	resp, err := client.GetApplication(ctx, *applicationId, application.DefaultGetApplicationOperationOptions())
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return tf.ErrorDiagPathF(nil, "application_object_id", "Application with object ID %q was not found", applicationId.ApplicationId)
		}
		return tf.ErrorDiagPathF(err, "application_object_id", "Retrieving application with object ID %q", applicationId.ApplicationId)
	}

	app := resp.Model
	if app == nil || app.Id == nil {
		return tf.ErrorDiagF(errors.New("nil application or application with nil ID was returned"), "API error retrieving %s", applicationId)
	}

	request := application.AddPasswordRequest{
		PasswordCredential: credential,
	}
	addPasswordResp, err := client.AddPassword(ctx, *applicationId, request, application.DefaultAddPasswordOperationOptions())
	if err != nil {
		return tf.ErrorDiagF(err, "Adding password for %s", applicationId)
	}

	newCredential := addPasswordResp.Model
	if newCredential == nil {
		return tf.ErrorDiagF(errors.New("nil credential received when adding password"), "API error adding password for %s", applicationId)
	}
	if newCredential.KeyId.IsNull() {
		return tf.ErrorDiagF(errors.New("nil or empty keyId received"), "API error adding password for %s", applicationId)
	}

	password := newCredential.SecretText.GetOrZero()
	if len(password) == 0 {
		return tf.ErrorDiagF(errors.New("nil or empty password received"), "API error adding password for %s", applicationId)
	}

	id := parse.NewCredentialID(applicationId.ApplicationId, "password", newCredential.KeyId.GetOrZero())

	// Wait for the credential to appear in the application manifest, this can take several minutes
	timeout, _ := ctx.Deadline()
	polledForCredential, err := (&pluginsdk.StateChangeConf{ //nolint:staticcheck
		Pending:                   []string{"Waiting"},
		Target:                    []string{"Done"},
		Timeout:                   time.Until(timeout),
		MinTimeout:                1 * time.Second,
		ContinuousTargetOccurence: 5,
		Refresh: func() (interface{}, string, error) {
			resp, err := client.GetApplication(ctx, *applicationId, application.DefaultGetApplicationOperationOptions())
			if err != nil {
				return nil, "Error", err
			}

			if resp.Model.PasswordCredentials != nil {
				for _, cred := range *resp.Model.PasswordCredentials {
					if strings.EqualFold(cred.KeyId.GetOrZero(), id.KeyId) {
						return &cred, "Done", nil
					}
				}
			}

			return nil, "Waiting", nil
		},
	}).WaitForStateContext(ctx)

	if err != nil {
		return tf.ErrorDiagF(err, "Waiting for password credential for %s", applicationId)
	} else if polledForCredential == nil {
		return tf.ErrorDiagF(errors.New("password credential not found in application manifest"), "Waiting for password credential for %s", applicationId)
	}

	d.SetId(id.String())
	d.Set("value", newCredential.SecretText.GetOrZero())

	return applicationPasswordResourceRead(ctx, d, meta)
}

func applicationPasswordResourceRead(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics { //nolint
	client := meta.(*clients.Client).Applications.ApplicationClient

	id, err := parse.PasswordID(d.Id())
	if err != nil {
		return tf.ErrorDiagPathF(err, "id", "Parsing password credential with ID %q", d.Id())
	}

	applicationId := stable.NewApplicationID(id.ObjectId)

	resp, err := client.GetApplication(ctx, applicationId, application.DefaultGetApplicationOperationOptions())
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[DEBUG] %s for %s credential %q was not found - removing from state!", applicationId, id.KeyType, id.KeyId)
			d.SetId("")
			return nil
		}
		return tf.ErrorDiagPathF(err, "application_object_id", "Retrieving %s", applicationId)
	}

	app := resp.Model
	if app == nil {
		return tf.ErrorDiagF(errors.New("model was nil"), "Retrieving %s", applicationId)
	}

	credential := credentials.GetPasswordCredential(app.PasswordCredentials, id.KeyId)
	if credential == nil {
		log.Printf("[DEBUG] Password credential %q (ID %q) was not found - removing from state!", id.KeyId, id.ObjectId)
		d.SetId("")
		return nil
	}

	tf.Set(d, "application_id", applicationId.ID())

	if credential.DisplayName != nil {
		tf.Set(d, "display_name", credential.DisplayName.GetOrZero())
	} else if credential.CustomKeyIdentifier != nil {
		displayName, err := base64.StdEncoding.DecodeString(credential.CustomKeyIdentifier.GetOrZero())
		if err != nil {
			return tf.ErrorDiagPathF(err, "display_name", "Parsing CustomKeyIdentifier")
		}
		tf.Set(d, "display_name", string(displayName))
	}

	tf.Set(d, "key_id", id.KeyId)
	tf.Set(d, "start_date", credential.StartDateTime.GetOrZero())
	tf.Set(d, "end_date", credential.EndDateTime.GetOrZero())

	if v := d.Get("application_object_id").(string); v != "" {
		tf.Set(d, "application_object_id", v)
	} else {
		tf.Set(d, "application_object_id", id.ObjectId)
	}

	return nil
}

func applicationPasswordResourceDelete(ctx context.Context, d *pluginsdk.ResourceData, meta interface{}) pluginsdk.Diagnostics { //nolint
	client := meta.(*clients.Client).Applications.ApplicationClient

	id, err := parse.PasswordID(d.Id())
	if err != nil {
		return tf.ErrorDiagPathF(err, "id", "Parsing password credential with ID %q", d.Id())
	}

	applicationId := stable.NewApplicationID(id.ObjectId)

	tf.LockByName(applicationResourceName, id.ObjectId)
	defer tf.UnlockByName(applicationResourceName, id.ObjectId)

	request := application.RemovePasswordRequest{
		KeyId: pointer.To(id.KeyId),
	}
	if _, err = client.RemovePassword(ctx, applicationId, request, application.DefaultRemovePasswordOperationOptions()); err != nil {
		return tf.ErrorDiagF(err, "Removing password credential %q from %s", id.KeyId, applicationId)
	}

	// Wait for application password to be deleted
	if err := consistency.WaitForDeletion(ctx, func(ctx context.Context) (*bool, error) {
		resp, err := client.GetApplication(ctx, applicationId, application.DefaultGetApplicationOperationOptions())
		if err != nil {
			return nil, err
		}

		app := resp.Model
		if app == nil {
			return nil, errors.New("model was nil")
		}

		return pointer.To(credentials.GetPasswordCredential(app.PasswordCredentials, id.KeyId) != nil), nil
	}); err != nil {
		return tf.ErrorDiagF(err, "Waiting for deletion of password credential %q from %s", id.KeyId, applicationId)
	}

	return nil
}
