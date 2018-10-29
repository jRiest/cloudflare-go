package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	"github.com/pkg/errors"
)

// WorkerRequestParams provides parameters for worker requests for both enterprise and standard requests
type WorkerRequestParams struct {
	ZoneID     string
	ScriptName string
}

type bindingBodyWriter func(*multipart.Writer) error

type WorkerBinding interface {
	serialize(string) (json.RawMessage, bindingBodyWriter, error)
}

// WorkerScriptParams provides a worker script and the associated bindings
type WorkerScriptParams struct {
	Script   string
	Bindings map[string]WorkerBinding
}

// WorkerRoute aka filters are patterns used to enable or disable workers that match requests.
//
// API reference: https://api.cloudflare.com/#worker-filters-properties
type WorkerRoute struct {
	ID      string `json:"id,omitempty"`
	Pattern string `json:"pattern"`
	Enabled bool   `json:"enabled"`
	Script  string `json:"script,omitempty"`
}

// WorkerRoutesResponse embeds Response struct and slice of WorkerRoutes
type WorkerRoutesResponse struct {
	Response
	Routes []WorkerRoute `json:"result"`
}

// WorkerRouteResponse embeds Response struct and a single WorkerRoute
type WorkerRouteResponse struct {
	Response
	WorkerRoute `json:"result"`
}

// WorkerScript Cloudflare Worker struct with metadata
type WorkerScript struct {
	WorkerMetaData
	Script string `json:"script"`
}

// WorkerMetaData contains worker script information such as size, creation & modification dates
type WorkerMetaData struct {
	ID         string    `json:"id,omitempty"`
	ETAG       string    `json:"etag,omitempty"`
	Size       int       `json:"size,omitempty"`
	CreatedOn  time.Time `json:"created_on,omitempty"`
	ModifiedOn time.Time `json:"modified_on,omitempty"`
}

// WorkerListResponse wrapper struct for API response to worker script list API call
type WorkerListResponse struct {
	Response
	WorkerList []WorkerMetaData `json:"result"`
}

// WorkerScriptResponse wrapper struct for API response to worker script calls
type WorkerScriptResponse struct {
	Response
	WorkerScript `json:"result"`
}

// Bindings

type WorkerInheritBinding struct {
	OldName string
}

func (b WorkerInheritBinding) serialize(name string) (json.RawMessage, bindingBodyWriter, error) {
	type meta struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		OldName string `json:"old_name,omitempty"`
	}

	metadata, err := json.Marshal(meta{
		Name:    name,
		Type:    "inherit",
		OldName: b.OldName,
	})
	if err != nil {
		return nil, nil, err
	}

	return metadata, nil, nil
}

// DeleteWorker deletes worker for a zone.
//
// API reference: https://api.cloudflare.com/#worker-script-delete-worker
func (api *API) DeleteWorker(requestParams *WorkerRequestParams) (WorkerScriptResponse, error) {
	// if ScriptName is provided we will treat as org request
	if requestParams.ScriptName != "" {
		return api.deleteWorkerWithName(requestParams.ScriptName)
	}
	uri := "/zones/" + requestParams.ZoneID + "/workers/script"
	res, err := api.makeRequest("DELETE", uri, nil)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return r, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

// DeleteWorkerWithName deletes worker for a zone.
// This is an enterprise only feature https://developers.cloudflare.com/workers/api/config-api-for-enterprise
// organizationID must be specified as api option https://godoc.org/github.com/cloudflare/cloudflare-go#UsingOrganization
//
// API reference: https://api.cloudflare.com/#worker-script-delete-worker
func (api *API) deleteWorkerWithName(scriptName string) (WorkerScriptResponse, error) {
	if api.OrganizationID == "" {
		return WorkerScriptResponse{}, errors.New("organization ID required for enterprise only request")
	}
	uri := "/accounts/" + api.OrganizationID + "/workers/scripts/" + scriptName
	res, err := api.makeRequest("DELETE", uri, nil)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return r, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

// DownloadWorker fetch raw script content for your worker returns []byte containing worker code js
//
// API reference: https://api.cloudflare.com/#worker-script-download-worker
func (api *API) DownloadWorker(requestParams *WorkerRequestParams) (WorkerScriptResponse, error) {
	if requestParams.ScriptName != "" {
		return api.downloadWorkerWithName(requestParams.ScriptName)
	}
	uri := "/zones/" + requestParams.ZoneID + "/workers/script"
	res, err := api.makeRequest("GET", uri, nil)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	r.Script = string(res)
	r.Success = true
	return r, nil
}

// DownloadWorkerWithName fetch raw script content for your worker returns string containing worker code js
// This is an enterprise only feature https://developers.cloudflare.com/workers/api/config-api-for-enterprise/
//
// API reference: https://api.cloudflare.com/#worker-script-download-worker
func (api *API) downloadWorkerWithName(scriptName string) (WorkerScriptResponse, error) {
	if api.OrganizationID == "" {
		return WorkerScriptResponse{}, errors.New("organization ID required for enterprise only request")
	}
	uri := "/accounts/" + api.OrganizationID + "/workers/scripts/" + scriptName
	res, err := api.makeRequest("GET", uri, nil)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	r.Script = string(res)
	r.Success = true
	return r, nil
}

// ListWorkerScripts returns list of worker scripts for given organization
// This is an enterprise only feature https://developers.cloudflare.com/workers/api/config-api-for-enterprise
//
// API reference: https://developers.cloudflare.com/workers/api/config-api-for-enterprise/
func (api *API) ListWorkerScripts() (WorkerListResponse, error) {
	if api.OrganizationID == "" {
		return WorkerListResponse{}, errors.New("organization ID required for enterprise only request")
	}
	uri := "/accounts/" + api.OrganizationID + "/workers/scripts"
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return WorkerListResponse{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WorkerListResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WorkerListResponse{}, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

// UploadWorker push raw script content for your worker
//
// API reference: https://api.cloudflare.com/#worker-script-upload-worker
func (api *API) UploadWorker(requestParams *WorkerRequestParams, data string) (WorkerScriptResponse, error) {
	if requestParams.ScriptName != "" {
		return api.multiScriptUpload(requestParams.ScriptName, "application/javascript", []byte(data))
	}
	return api.singleScriptUpload(requestParams.ZoneID, "application/javascript", []byte(data))
}

// UploadWorkerWithBindings push raw script content and bindings for your worker
//
// API reference: https://api.cloudflare.com/#worker-script-upload-worker
func (api *API) UploadWorkerWithBindings(requestParams *WorkerRequestParams, data WorkerScriptParams) (WorkerScriptResponse, error) {
	contentType, body, err := formatMultipartBody(data)
	if err != nil {
		return WorkerScriptResponse{}, err
	}
	if requestParams.ScriptName != "" {
		return api.multiScriptUpload(requestParams.ScriptName, contentType, body)
	}
	return api.singleScriptUpload(requestParams.ZoneID, contentType, body)
}

func (api *API) singleScriptUpload(zoneId, contentType string, body []byte) (WorkerScriptResponse, error) {
	uri := "/zones/" + zoneId + "/workers/script"
	headers := make(http.Header)
	headers.Set("Content-Type", contentType)
	res, err := api.makeRequestWithHeaders("PUT", uri, body, headers)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return r, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

func (api *API) multiScriptUpload(scriptName, contentType string, body []byte) (WorkerScriptResponse, error) {
	if api.OrganizationID == "" {
		return WorkerScriptResponse{}, errors.New("organization ID required for enterprise only request")
	}
	uri := "/accounts/" + api.OrganizationID + "/workers/scripts/" + scriptName
	headers := make(http.Header)
	headers.Set("Content-Type", contentType)
	res, err := api.makeRequestWithHeaders("PUT", uri, body, headers)
	var r WorkerScriptResponse
	if err != nil {
		return r, errors.Wrap(err, errMakeRequestError)
	}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return r, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

func hasKey(m map[string]WorkerBinding, key string) bool {
	_, hasKey := m[key]
	return hasKey
}

// Returns content-type, body, error
func formatMultipartBody(params WorkerScriptParams) (string, []byte, error) {
	var buf = &bytes.Buffer{}
	var mpw = multipart.NewWriter(buf)
	defer mpw.Close()

	// Write metadata part
	type metadata struct {
		BodyPart string            `json:"body_part"`
		Bindings []json.RawMessage `json:"bindings"`
	}

	scriptBodyPart := "script"
	for hasKey(params.Bindings, scriptBodyPart) {
		// Make sure that the script body part doesn't collide with any binding body parts
		scriptBodyPart = scriptBodyPart + "_"
	}
	meta := metadata{
		BodyPart: scriptBodyPart,
		Bindings: make([]json.RawMessage, 0, len(params.Bindings)),
	}

	bodyWriters := make([]bindingBodyWriter, 0, len(params.Bindings))
	for name, b := range params.Bindings {
		bindingMeta, bodyWriter, err := b.serialize(name)
		if err != nil {
			return "", nil, err
		}

		meta.Bindings = append(meta.Bindings, bindingMeta)
		bodyWriters = append(bodyWriters, bodyWriter)
	}

	var hdr = textproto.MIMEHeader{}
	hdr.Set("content-disposition", fmt.Sprintf(`form-data; name="%s"`, "metadata"))
	hdr.Set("content-type", "application/json")
	pw, err := mpw.CreatePart(hdr)
	if err != nil {
		return "", nil, err
	}
	metaJson, err := json.Marshal(meta)
	if err != nil {
		return "", nil, err
	}
	_, err = pw.Write(metaJson)
	if err != nil {
		return "", nil, err
	}

	// Write script part
	hdr = textproto.MIMEHeader{}
	hdr.Set("content-disposition", fmt.Sprintf(`form-data; name="%s"`, scriptBodyPart))
	hdr.Set("content-type", "application/javascript")
	pw, err = mpw.CreatePart(hdr)
	_, err = pw.Write([]byte(params.Script))
	if err != nil {
		return "", nil, err
	}

	// Write other bindings with parts
	for _, w := range bodyWriters {
		if w != nil {
			err = w(mpw)
			if err != nil {
				return "", nil, err
			}
		}
	}

	mpw.Close()

	return mpw.FormDataContentType(), buf.Bytes(), nil
}

// CreateWorkerRoute creates worker route for a zone
//
// API reference: https://api.cloudflare.com/#worker-filters-create-filter
func (api *API) CreateWorkerRoute(zoneID string, route WorkerRoute) (WorkerRouteResponse, error) {
	// Check whether a script name is defined in order to determine whether
	// to use the single-script or multi-script endpoint.
	pathComponent := "filters"
	if route.Script != "" {
		if api.OrganizationID == "" {
			return WorkerRouteResponse{}, errors.New("organization ID required for enterprise only request")
		}
		pathComponent = "routes"
	}

	uri := "/zones/" + zoneID + "/workers/" + pathComponent
	res, err := api.makeRequest("POST", uri, route)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WorkerRouteResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

// DeleteWorkerRoute deletes worker route for a zone
//
// API reference: https://api.cloudflare.com/#worker-filters-delete-filter
func (api *API) DeleteWorkerRoute(zoneID string, routeID string) (WorkerRouteResponse, error) {
	// For deleting a route, it doesn't matter whether we use the
	// single-script or multi-script endpoint
	uri := "/zones/" + zoneID + "/workers/filters/" + routeID
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WorkerRouteResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}

// ListWorkerRoutes returns list of worker routes
//
// API reference: https://api.cloudflare.com/#worker-filters-list-filters
func (api *API) ListWorkerRoutes(zoneID string) (WorkerRoutesResponse, error) {
	pathComponent := "filters"
	if api.OrganizationID != "" {
		pathComponent = "routes"
	}
	uri := "/zones/" + zoneID + "/workers/" + pathComponent
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return WorkerRoutesResponse{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WorkerRoutesResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WorkerRoutesResponse{}, errors.Wrap(err, errUnmarshalError)
	}
	for i := range r.Routes {
		route := &r.Routes[i]
		// The Enabled flag will not be set in the multi-script API response
		// so we manually set it to true if the script name is not empty
		// in case any multi-script customers rely on the Enabled field
		if route.Script != "" {
			route.Enabled = true
		}
	}
	return r, nil
}

// UpdateWorkerRoute updates worker route for a zone.
//
// API reference: https://api.cloudflare.com/#worker-filters-update-filter
func (api *API) UpdateWorkerRoute(zoneID string, routeID string, route WorkerRoute) (WorkerRouteResponse, error) {
	// Check whether a script name is defined in order to determine whether
	// to use the single-script or multi-script endpoint.
	pathComponent := "filters"
	if route.Script != "" {
		if api.OrganizationID == "" {
			return WorkerRouteResponse{}, errors.New("organization ID required for enterprise only request")
		}
		pathComponent = "routes"
	}
	uri := "/zones/" + zoneID + "/workers/" + pathComponent + "/" + routeID
	res, err := api.makeRequest("PUT", uri, route)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errMakeRequestError)
	}
	var r WorkerRouteResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return WorkerRouteResponse{}, errors.Wrap(err, errUnmarshalError)
	}
	return r, nil
}
