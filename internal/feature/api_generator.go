package feature

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/getkin/kin-openapi/openapi3"
)

type APIGenerator struct {
	spec *openapi3.T
}

func NewAPIGenerator() *APIGenerator {
	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "WAWA",
			Version:     "1.0.0",
			Description: "API documentation for WAWA",
		},
		Servers: []*openapi3.Server{
			{
				URL:         "https://api.coreconcept.tech",
				Description: "PRODUCTION",
			},
		},
		Paths: &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: make(map[string]*openapi3.SchemaRef),
			SecuritySchemes: map[string]*openapi3.SecuritySchemeRef{
				"apiKey": {
					Value: &openapi3.SecurityScheme{
						Type:        "apiKey",
						In:          "header",
						Name:        cfg.Default().Site.HTTPHeaderUserAccessTokenKey,
						Description: "API Key authentication",
					},
				},
			},
		},
	}
	return &APIGenerator{
		spec: spec,
	}
}

// AddEndpoint adds an endpoint to the OpenAPI spec from a RequestObject
func (g *APIGenerator) AddEndpoint(requestObj any, responseType reflect.Type) error {
	// Get API settings using reflection
	settingsMethod := reflect.ValueOf(requestObj).MethodByName("APISettings")
	if !settingsMethod.IsValid() {
		return fmt.Errorf("APISettings method not found")
	}
	settingsResult := settingsMethod.Call(nil)
	if len(settingsResult) == 0 {
		return fmt.Errorf("APISettings method returned no results")
	}
	settings, ok := settingsResult[0].Interface().(APISettings)
	if !ok {
		return fmt.Errorf("APISettings method did not return APISettings type")
	}
	// Create operation
	operation := &openapi3.Operation{
		Summary:     settings.Summary,
		Description: settings.Description,
		Responses:   openapi3.NewResponses(),
		Parameters:  []*openapi3.ParameterRef{},
	}

	// Add tag if specified
	if settings.Tag != "" {
		operation.Tags = []string{string(settings.Tag)}
	}
	// Add security if auth is required
	if settings.Auth {
		operation.Security = &openapi3.SecurityRequirements{
			{
				"apiKey": []string{},
			},
		}
	}
	// Extract path parameters from URI tags
	pathParams := g.extractPathParameters(reflect.TypeOf(requestObj))
	for _, param := range pathParams {
		operation.Parameters = append(operation.Parameters, param)
	}

	// Extract query parameters from form tags
	queryParams := g.extractQueryParameters(reflect.TypeOf(requestObj))
	for _, param := range queryParams {
		operation.Parameters = append(operation.Parameters, param)
	}

	// Generate request body schema if needed.
	// Some endpoints read raw binary data from the request body even though they also
	// include query/form parameters, so we must allow explicit binary request bodies.
	hasBody := settings.Type != types.HttpRequestTypeUri &&
		settings.Type != types.HttpRequestTypeUriQuery
	if settings.Type == types.HttpRequestTypeQuery && settings.BodyContentType == "application/json" {
		hasBody = false
	}

	if (settings.Method == "POST" || settings.Method == "PUT" || settings.Method == "PATCH") && hasBody {
		requestSchema := g.generateRequestBodySchema(reflect.TypeOf(requestObj))
		if requestSchema == nil && settings.BodyContentType == "application/octet-stream" {
			requestSchema = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "binary"}}
		}
		if requestSchema != nil {
			contentType := settings.BodyContentType
			if contentType == "" {
				contentType = "application/json"
			}
			operation.RequestBody = &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: map[string]*openapi3.MediaType{
						contentType: {
							Schema: requestSchema,
						},
					},
				},
			}
		}
	}
	// Generate response schema
	if settings.WebSocket {
		operation.Responses.Set(fmt.Sprint(http.StatusSwitchingProtocols), &openapi3.ResponseRef{
			Value: &openapi3.Response{Description: helper.ConvertToPointer("WebSocket connection established")},
		})
	}
	if responseType != nil && !settings.WebSocket {
		successResponseSchema := g.generateResponseSchema(responseType, true, http.StatusOK, "")
		operation.Responses.Set(fmt.Sprint(http.StatusOK), &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: helper.ConvertToPointer("Successful response"),
				Content: map[string]*openapi3.MediaType{
					"application/json": {
						Schema: successResponseSchema,
					},
				},
			},
		})
		// generate possible errors
		// add default errors
		settings.Errors = append(settings.Errors, NewAPIError(*exception.NewException(types.ExceptionTypeInvalidInput)))
		settings.Errors = append(settings.Errors, NewAPIError(*exception.NewException(types.ExceptionTypeInternalServer)))
		if settings.Auth {
			settings.Errors = append(settings.Errors, NewAPIError(*exception.NewException(types.ExceptionTypeUnauthorized)))
		}
		groups := helper.Group(settings.Errors, func(e APIError) int {
			return e.StatusCode
		})
		for key, group := range groups {
			messages := helper.Map(group, func(e APIError) string {
				return e.Description
			})
			message := strings.Join(messages, ", ")
			errorResponseSchema := g.generateResponseSchema(responseType, false, key.(int), message)
			operation.Responses.Set(fmt.Sprint(key.(int)), &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: &message,
					Content: map[string]*openapi3.MediaType{
						"application/json": {
							Schema: errorResponseSchema,
						},
					},
				},
			})
		}
	}

	// Add to paths - handle multiple HTTP methods on same path
	if g.spec.Paths == nil {
		g.spec.Paths = &openapi3.Paths{}
	}
	// Convert Gin-style :param to OpenAPI {param} format reliably
	path := settings.Path
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if strings.HasPrefix(seg, ":") {
			segs[i] = "{" + seg[1:] + "}"
		}
	}
	path = strings.Join(segs, "/")

	// Get existing path item or create new one
	var pathItem *openapi3.PathItem
	if existingPathItem := g.spec.Paths.Find(path); existingPathItem != nil {
		pathItem = existingPathItem
	} else {
		pathItem = &openapi3.PathItem{}
	}

	// Set operation based on method
	switch strings.ToUpper(settings.Method) {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	}

	// Set the updated path item
	g.spec.Paths.Set(path, pathItem)

	// Add tag to the spec if it doesn't exist
	if settings.Tag != "" {
		g.addTagToSpec(string(settings.Tag))
	}

	return nil
}

// addTagToSpec adds a tag definition to the OpenAPI spec
func (g *APIGenerator) addTagToSpec(tagName string) {
	if g.spec.Tags == nil {
		g.spec.Tags = openapi3.Tags{}
	}

	// Check if tag already exists
	for _, existingTag := range g.spec.Tags {
		if existingTag.Name == tagName {
			return // Tag already exists
		}
	}

	// Add tag if it doesn't exist
	g.spec.Tags = append(g.spec.Tags, &openapi3.Tag{
		Name: tagName,
	})
}

// generateRequestBodySchema creates an OpenAPI schema from a Go struct for request bodies, excluding URI parameters
func (g *APIGenerator) generateRequestBodySchema(t reflect.Type) *openapi3.SchemaRef {
	return g.generateSchemaRecursive(t, make(map[reflect.Type]bool), 0, true)
}

// generateSchemaFromStruct creates an OpenAPI schema from a Go struct
func (g *APIGenerator) generateSchemaFromStruct(t reflect.Type) *openapi3.SchemaRef {
	return g.generateSchemaRecursive(t, make(map[reflect.Type]bool), 0, false)
}

func (g *APIGenerator) generateSchemaRecursive(t reflect.Type, visited map[reflect.Type]bool, depth int, isRequest bool) *openapi3.SchemaRef {
	if depth > 10 {
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if visited[t] {
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: &openapi3.Types{"object"},
			},
		}
	}
	visited[t] = true
	defer delete(visited, t)

	// Handle special types before checking if it's a struct
	if t.String() == "time.Time" {
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "date-time",
			},
		}
	}
	if t.PkgPath() == "github.com/google/uuid" && t.Name() == "UUID" {
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "uuid",
			},
		}
	}

	if t.Kind() != reflect.Struct {
		return g.generatePrimitiveSchemaRecursive(t, visited, depth)
	}

	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(map[string]*openapi3.SchemaRef),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs (anonymous fields)
		if field.Anonymous {
			embeddedSchema := g.generateSchemaRecursive(field.Type, visited, depth+1, isRequest)
			if embeddedSchema != nil && embeddedSchema.Value != nil {
				maps.Copy(schema.Properties, embeddedSchema.Value.Properties)
				schema.Required = append(schema.Required, embeddedSchema.Value.Required...)
			}
			continue
		}

		if isRequest {
			// Skip path/query params for request body
			if field.Tag.Get("uri") != "" || field.Tag.Get("form") != "" {
				continue
			}
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		isOptional := strings.Contains(jsonTag, "omitempty")

		fieldSchema := g.generateSchemaRecursive(field.Type, visited, depth+1, isRequest)
		if fieldSchema != nil {
			schema.Properties[jsonName] = fieldSchema
			if valTag := field.Tag.Get("val"); valTag != "" {
				if strings.Contains(valTag, "required") && !isOptional {
					schema.Required = append(schema.Required, jsonName)
				} else if isRequest {
					fieldSchema.Value.Nullable = true
				}
			} else if isRequest {
				fieldSchema.Value.Nullable = true
			}

			if descTag := field.Tag.Get("description"); descTag != "" {
				if fieldSchema.Value != nil {
					fieldSchema.Value.Description = descTag
				}
			}
			if exampleTag := field.Tag.Get("example"); exampleTag != "" {
				if fieldSchema.Value != nil {
					fieldSchema.Value.Example = g.convertExampleValue(exampleTag, fieldSchema)
				}
			}
		}
	}
	return &openapi3.SchemaRef{Value: schema}
}

// generatePrimitiveSchemaRecursive creates schema for primitive types with recursion depth
func (g *APIGenerator) generatePrimitiveSchemaRecursive(t reflect.Type, visited map[reflect.Type]bool, depth int) *openapi3.SchemaRef {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	schema := &openapi3.Schema{}
	switch t.Kind() {
	case reflect.String:
		schema.Type = &openapi3.Types{"string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = &openapi3.Types{"integer"}
		if t.Kind() == reflect.Int64 {
			schema.Format = "int64"
		} else {
			schema.Format = "int32"
		}
	case reflect.Float32, reflect.Float64:
		schema.Type = &openapi3.Types{"number"}
		if t.Kind() == reflect.Float32 {
			schema.Format = "float"
		} else {
			schema.Format = "double"
		}
	case reflect.Bool:
		schema.Type = &openapi3.Types{"boolean"}
	case reflect.Slice, reflect.Array:
		schema.Type = &openapi3.Types{"array"}
		schema.Items = g.generateSchemaRecursive(t.Elem(), visited, depth+1, false)
	case reflect.Map:
		schema.Type = &openapi3.Types{"object"}
		schema.AdditionalProperties = openapi3.AdditionalProperties{
			Schema: g.generateSchemaRecursive(t.Elem(), visited, depth+1, false),
		}
	case reflect.Struct:
		if t.String() == "time.Time" {
			schema.Type = &openapi3.Types{"string"}
			schema.Format = "date-time"
		} else if t.PkgPath() == "github.com/google/uuid" && t.Name() == "UUID" {
			schema.Type = &openapi3.Types{"string"}
			schema.Format = "uuid"
		} else {
			return g.generateSchemaRecursive(t, visited, depth, false)
		}
	default:
		schema.Type = &openapi3.Types{"object"}
	}
	return &openapi3.SchemaRef{Value: schema}
}

// existing methods below updated to call recursive counterparts
func (g *APIGenerator) generatePrimitiveSchema(t reflect.Type) *openapi3.SchemaRef {
	return g.generatePrimitiveSchemaRecursive(t, make(map[reflect.Type]bool), 0)
}

// generateResponseSchema creates the response schema using reflection on dto.ResponseObject
func (g *APIGenerator) generateResponseSchema(responseType reflect.Type, success bool, statusCode int, message string) *openapi3.SchemaRef {
	// Create a dummy ResponseObject to analyze its structure
	var dummyResponse dto.Response[any]
	responseObjectType := reflect.TypeOf(dummyResponse)
	// Generate the base ResponseObject schema using reflection
	baseSchema := g.generateSchemaFromStruct(responseObjectType)

	// Now customize the data field with the actual response type
	if success && baseSchema != nil && baseSchema.Value != nil && responseType != nil && responseType.Kind() != reflect.Invalid {
		dataSchema := g.generateSchemaFromStruct(responseType)
		if dataSchema != nil {
			// Preserve description from base schema's data field if it exists
			if oldData, ok := baseSchema.Value.Properties["data"]; ok && oldData.Value != nil {
				if dataSchema.Value != nil && dataSchema.Value.Description == "" {
					dataSchema.Value.Description = oldData.Value.Description
				}
			}
			baseSchema.Value.Properties["data"] = dataSchema
		}
	}

	// Update common fields while preserving metadata like descriptions
	if baseSchema != nil && baseSchema.Value != nil {
		if prop, ok := baseSchema.Value.Properties["success"]; ok && prop.Value != nil {
			prop.Value.Example = success
		}
		if prop, ok := baseSchema.Value.Properties["status_code"]; ok && prop.Value != nil {
			prop.Value.Example = statusCode
		}
		if prop, ok := baseSchema.Value.Properties["message"]; ok && prop.Value != nil {
			prop.Value.Example = message
		}

		if success {
			delete(baseSchema.Value.Properties, "input_errors")
			// Only delete message if it's empty in success case
			if message == "" {
				delete(baseSchema.Value.Properties, "message")
			}
			// If response type is empty (any/interface{}), remove data field
			isEmpty := responseType == nil || responseType.Kind() == reflect.Invalid ||
				(responseType.Kind() == reflect.Interface && responseType.NumMethod() == 0)
			if isEmpty {
				delete(baseSchema.Value.Properties, "data")
			}
		} else {
			delete(baseSchema.Value.Properties, "data")
			if statusCode != 400 {
				delete(baseSchema.Value.Properties, "input_errors")
			}
		}
	}

	return baseSchema
}

// GenerateJSON returns the OpenAPI spec as JSON
func (g *APIGenerator) GenerateJSON() ([]byte, error) {
	return json.MarshalIndent(g.spec, "", "  ")
}

// GenerateYAML returns the OpenAPI spec as YAML
func (g *APIGenerator) GenerateYAML() ([]byte, error) {
	return g.spec.MarshalJSON()
}

// convertExampleValue converts a string example to the appropriate type based on the schema
func (g *APIGenerator) convertExampleValue(example string, paramSchema *openapi3.SchemaRef) interface{} {
	if example == "" || paramSchema == nil || paramSchema.Value == nil {
		return example
	}

	if paramSchema.Value.Type == nil || len(*paramSchema.Value.Type) == 0 {
		return example
	}

	switch (*paramSchema.Value.Type)[0] {
	case "integer":
		if intVal, err := strconv.Atoi(example); err == nil {
			return intVal
		}
	case "number":
		if floatVal, err := strconv.ParseFloat(example, 64); err == nil {
			return floatVal
		}
	case "boolean":
		if boolVal, err := strconv.ParseBool(example); err == nil {
			return boolVal
		}
	}
	return example
}

// extractParameters extracts parameters from struct fields with the specified tag
func (g *APIGenerator) extractParameters(t reflect.Type, tagName string, parameterIn string) []*openapi3.ParameterRef {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var parameters []*openapi3.ParameterRef

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs (anonymous fields)
		if field.Anonymous {
			embeddedParams := g.extractParameters(field.Type, tagName, parameterIn)
			parameters = append(parameters, embeddedParams...)
			continue
		}

		tagValue := field.Tag.Get(tagName)
		if tagValue == "" {
			continue
		}

		// Create parameter schema based on field type
		paramSchema := g.generatePrimitiveSchema(field.Type)

		// Get description and example from tags
		description := field.Tag.Get("description")
		example := field.Tag.Get("example")

		// Determine if parameter is required
		// Path parameters are always required per OpenAPI 3.0 spec
		required := parameterIn == "path"
		if !required {
			if valTag := field.Tag.Get("val"); valTag != "" {
				required = strings.Contains(valTag, "required")
			}
		}

		// Convert example value to appropriate type
		exampleValue := g.convertExampleValue(example, paramSchema)

		parameter := &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        tagValue,
				In:          parameterIn,
				Required:    required,
				Description: description,
				Schema:      paramSchema,
				Example:     exampleValue,
			},
		}

		parameters = append(parameters, parameter)
	}

	return parameters
}

// extractPathParameters extracts path parameters from struct fields with uri tags
func (g *APIGenerator) extractPathParameters(t reflect.Type) []*openapi3.ParameterRef {
	return g.extractParameters(t, "uri", "path")
}

// extractQueryParameters extracts query parameters from struct fields with form tags
func (g *APIGenerator) extractQueryParameters(t reflect.Type) []*openapi3.ParameterRef {
	return g.extractParameters(t, "form", "query")
}

// GetSpec returns the OpenAPI spec
func (g *APIGenerator) GetSpec() *openapi3.T {
	return g.spec
}
