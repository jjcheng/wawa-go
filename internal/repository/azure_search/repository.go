package azure_search

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
)

type Repository[T dto.SearchResponse] struct {
	baseUrl                   string
	apiVersion                string
	header                    map[string]string
	logger                    *service.Logger
	indexName                 string
	semanticConfigurationName string
	entityName                string
}

func NewRepository[T dto.SearchResponse](indexName string, semanticConfigurationName string, logger *service.Logger) repository.SearchRepository[T] {
	azureSearchRepository := Repository[T]{
		baseUrl:    cfg.Default().AzureSearch.BaseUrl,
		apiVersion: "2026-08-01-preview",
		header: map[string]string{
			"api-key": cfg.Default().AzureSearch.APIKey,
		},
		indexName:                 indexName,
		logger:                    logger,
		semanticConfigurationName: semanticConfigurationName,
	}
	var object T
	t := reflect.TypeOf(object)
	azureSearchRepository.entityName = t.Name()
	return &azureSearchRepository
}

// index
func (repository *Repository[T]) GetIndexes(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%v/indexes?$select=*&api-version=%v", repository.baseUrl, repository.apiVersion)
	status, response, _, err := helper.RequestHTTP(ctx, url, "GET", &repository.header, nil)
	if status == 200 {
		dic, err := helper.DeserializeJSON[map[string]any](*response)
		if err != nil {
			return nil, err
		}
		list := []string{}
		if values, ok := (*dic)["value"].([]any); ok && len(values) > 0 {
			for _, value := range values {
				if valueDic, ok := value.(map[string]any); ok {
					if name, ok := valueDic["name"].(string); ok {
						list = append(list, name)
					}
				}
			}
		}
		return list, nil
	}
	if err != nil {
		return nil, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, err
	}
	return nil, nil
}

func (repository *Repository[T]) GetIndex(ctx context.Context) (map[string]any, error) {
	url := fmt.Sprintf("%v/indexes('%v')?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	status, response, _, err := helper.RequestHTTP(ctx, url, "GET", &repository.header, nil)
	if status == 200 {
		dic, err := helper.DeserializeJSON[map[string]any](*response)
		if err != nil {
			return nil, err
		}
		return *dic, nil
	}
	if err != nil {
		return nil, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, err
	}
	return nil, nil
}

func (repository *Repository[T]) Count(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%v/indexes('%v')/docs/$count?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	status, response, _, err := helper.RequestHTTP(ctx, url, "GET", &repository.header, nil)
	if status == 200 {
		i, err := strconv.Atoi(*response)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	if err != nil {
		return 0, err
	}
	if err := repository.handleError(response, status); err != nil {
		return 0, err
	}
	return 0, nil
}

func (repository *Repository[T]) DeleteIndex(ctx context.Context) error {
	url := fmt.Sprintf("%v/indexes('%v')?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	status, response, _, err := helper.RequestHTTP(ctx, url, "DELETE", &repository.header, nil)
	if status == 204 {
		return nil
	}
	if err != nil {
		return err
	}
	if err := repository.handleError(response, status); err != nil {
		return err
	}
	return nil
}

// point
func (repository *Repository[T]) GetById(ctx context.Context, id string, returnEmbedding bool) (*T, error, bool) {
	items, err := repository.GetByIds(ctx, []string{id}, returnEmbedding)
	if err != nil {
		return nil, err, false
	}
	if len(items) == 0 {
		return nil, nil, true
	}
	return &items[0], nil, false
}

func (repository *Repository[T]) GetByIds(ctx context.Context, ids []string, returnEmbedding bool) ([]T, error) {
	points, err := repository.GetPointsByIds(ctx, ids, returnEmbedding)
	if err != nil {
		return nil, err
	}
	return points, nil
}

// get points; if returnFields is nil, will return all fields. orderby is mandatory, use "entry_date" for default
func (repository *Repository[T]) GetAllPoints(ctx context.Context, filter string, orderBy string, desc bool, returnEmbedding bool, allPoints *[]T) error {
	nextSkip := 0
	for nextSkip > 0 || len(*allPoints) == 0 {
		points, ns, totalCount, err := repository.FilterPoints(ctx, filter, 0, nextSkip, orderBy, desc, returnEmbedding)
		if err != nil {
			return err
		}
		if totalCount == 0 {
			return nil
		}
		*allPoints = append(*allPoints, points...)
		nextSkip = ns
	}
	return nil
}

// will return the nextSkip for pagination, max is 1000, orderby is mandatory, use "entry_date" for default
func (repository *Repository[T]) FilterPoints(ctx context.Context, filter string, limit int, skip int, orderBy string, desc bool, returnEmbedding bool) (points []T, nextSkip int, totalCount int, err error) {
	if limit > 1000 {
		return nil, 0, 0, errors.New("max limit is 1000, if more use getAllPoints")
	}
	return repository.filterPoints(ctx, filter, limit, skip, orderBy, desc, returnEmbedding)
}

// orderby is mandatory, use "entry_date" for default
func (repository *Repository[T]) filterPoints(ctx context.Context, filter string, limit int, skip int, orderBy string, desc bool, withEmbedding bool) (points []T, nextSkip int, totalCount int, err error) {
	if limit == 0 && orderBy == "" {
		err := errors.New("limit is 0 and missing orderby")
		return nil, 0, 0, err
	}
	var requestObject T
	url := fmt.Sprintf("%v/indexes('%v')/docs/search.post.search?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		//"search": "*",
		"count": true,
		"skip":  skip,
	}
	returnFields := requestObject.ReturnFields(withEmbedding)
	if len(returnFields) > 0 {
		body["select"] = strings.Join(returnFields, ", ")
	} else {
		body["select"] = "*"
	}
	if limit > 0 {
		body["top"] = limit
	}
	if filter != "" {
		body["filter"] = filter
	}
	if orderBy != "" {
		order := orderBy
		if desc {
			order += " desc"
		}
		body["orderby"] = order
	} else {
		body["orderby"] = "entry_date desc"
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if status == 200 && response != nil {
		dic, err := helper.DeserializeJSON[map[string]any](*response)
		if err != nil {
			return nil, 0, 0, err
		}
		values := (*dic)["value"].([]any)
		points := []T{}
		var requestObject T
		for _, value := range values {
			payload := value.(map[string]any)
			object, err := requestObject.NewObject(payload)
			if err != nil {
				return nil, 0, 0, err
			}
			points = append(points, object.(T))
		}
		totalCount := 0
		if tc, ok := (*dic)["@odata.count"].(float64); ok {
			totalCount = int(tc)
		}
		if next, ok := (*dic)["@search.nextPageParameters"].(map[string]any); ok {
			if skip, ok := next["skip"].(float64); ok {
				nextSkip = int(skip)
			}
		} else if len(points)+skip < totalCount { //if skip + points count = total count, no more next
			nextSkip = skip + len(points)
		}
		return points, nextSkip, totalCount, nil
	}
	if err != nil {
		return nil, 0, 0, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, 0, 0, err
	}
	return nil, 0, 0, nil
}

func (repository *Repository[T]) GetPoint(ctx context.Context, id string, returnEmbedding bool) (*T, error, bool) {
	filter := fmt.Sprintf("id eq '%v'", id)
	points, _, _, err := repository.FilterPoints(ctx, filter, 1, 0, "entry_date", false, returnEmbedding)
	if err != nil {
		return nil, err, false
	}
	if len(points) == 0 {
		return nil, nil, true
	}
	return &points[0], nil, false
}

func (repository *Repository[T]) GetPointsByIds(ctx context.Context, ids []string, returnEmbedding bool) ([]T, error) {
	filter := fmt.Sprintf("search.in(id, '%v', ',')", strings.Join(ids, ","))
	allPoints := []T{}
	ex := repository.GetAllPoints(ctx, filter, "entry_date", false, returnEmbedding, &allPoints)
	if ex != nil {
		return nil, ex
	}
	return allPoints, nil
}

// CRUD
func (repository *Repository[T]) UpsertPoint(ctx context.Context, point T) error {
	return repository.UpsertPoints(ctx, []T{point}, 1)
}

func (repository *Repository[T]) UpsertPoints(ctx context.Context, points []T, batchSize int) error {
	pages := int(math.Ceil(float64(len(points)) / float64(batchSize)))
	if pages > 1 {
		repository.logger.Debugf("total %d pages", pages)
	}
	for i := 0; i <= pages-1; i++ {
		if pages > 1 {
			repository.logger.Debugf("page %d", i+1)
		}
		var pagePoints []T
		if i == pages-1 {
			pagePoints = points[i*batchSize:]
		} else {
			pagePoints = points[i*batchSize : i*batchSize+batchSize]
		}
		err := repository.upsertPoints(ctx, pagePoints)
		if err != nil {
			return err
		}
	}
	return nil
}

func (repository *Repository[T]) upsertPoints(ctx context.Context, points []T) error {
	url := fmt.Sprintf("%v/indexes/%v/docs/index?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	values := []map[string]any{}
	for _, point := range points {
		value := map[string]any{
			"@search.action": "upload",
		}
		payload, err := point.Payload()
		if err != nil {
			return err
		}
		maps.Copy(value, payload)
		values = append(values, value)
	}
	body := map[string]any{
		"value": values,
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return err
	}
	if err := repository.handleError(response, status); err != nil {
		return err
	}
	return nil
}

func (repository *Repository[T]) UpdateFields(ctx context.Context, id string, payload map[string]any) error {
	url := fmt.Sprintf("%v/indexes/%v/docs/index?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	payload["@search.action"] = "merge"
	payload["id"] = id
	body := map[string]any{
		"value": []map[string]any{payload},
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return err
	}
	if err := repository.handleError(response, status); err != nil {
		return err
	}
	return nil
}

func (repository *Repository[T]) DeletePoint(ctx context.Context, id string) error {
	url := fmt.Sprintf("%v/indexes/%v/docs/index?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"value": []map[string]any{
			{
				"@search.action": "delete",
				"id":             id,
			},
		},
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return err
	}
	if err := repository.handleError(response, status); err != nil {
		return err
	}
	return nil
}

func (repository *Repository[T]) DeletePoints(ctx context.Context, filter string) error {
	allPoints := []T{}
	err := repository.GetAllPoints(ctx, filter, "entry_date", false, false, &allPoints)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%v/indexes/%v/docs/index?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	values := []map[string]any{}
	for _, point := range allPoints {
		value := map[string]any{
			"@search.action": "delete",
			"id":             point.Base().Id,
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil
	}
	body := map[string]any{
		"value": values,
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return err
	}
	if err := repository.handleError(response, status); err != nil {
		return err
	}
	dic, err := helper.DeserializeJSON[map[string]any](*response)
	if err != nil {
		return err
	}
	errs := []string{}
	if values, ok := (*dic)["value"].([]any); ok {
		for _, value := range values {
			if valueDic, ok := value.(map[string]any); ok {
				if errorMessage, ok := valueDic["errorMessage"].(string); ok && errorMessage != "" {
					if id, ok := valueDic["id"].(string); ok {
						errs = append(errs, fmt.Sprintf("%v: %v", id, errorMessage))
					} else {
						errs = append(errs, fmt.Sprintf("%v: %v", "*", errorMessage))
					}
				}
			}
		}
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
		return err
	}
	return nil
}

func (repository *Repository[T]) preprocessQuery(query string) (processedQuery string, shouldNotSearch bool) {
	//https://learn.microsoft.com/en-gb/azure/search/query-lucene-syntax
	unsafeCharacters := []string{`+`, `-`, `&`, `|`, `!`, `(`, `)`, `{`, `}`, `[`, `]`, `^`, `"`, `~`, `*`, `?`, `:`, `\`, `/`}
	for _, unsafeCharacter := range unsafeCharacters {
		query = strings.ReplaceAll(query, unsafeCharacter, fmt.Sprintf("\\%v", unsafeCharacter))
	}
	//if the query has nothing else but the unsafe characters
	runes := strings.Split(query, "")
	if !helper.Any(runes, func(s string) bool {
		return !helper.Any(unsafeCharacters, func(r string) bool {
			return r == s
		})
	}) {
		return "", true
	}
	return query, false
}

// max query length 50
func (repository *Repository[T]) Suggest(ctx context.Context, query string, topK int, fuzzy bool, suggesterName string, searchFields []string, filter string, returnFields []string) ([]T, error) {
	query, shouldNotSearch := repository.preprocessQuery(query)
	if shouldNotSearch {
		return nil, nil
	}
	//suggest length cannot be more than 100 characters
	if len(query) > 100 {
		return nil, nil
	}
	url := fmt.Sprintf("%v/indexes('%v')/docs/search.post.suggest?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"search":           query,
		"suggesterName":    suggesterName,
		"fuzzy":            fuzzy,
		"highlightPreTag":  "<em>",
		"highlightPostTag": "</em>",
		"minimumCoverage":  80,
		"searchFields":     strings.Join(searchFields, ","),
		"top":              topK,
		"select":           strings.Join(returnFields, ", "),
	}
	if filter != "" {
		body["filter"] = filter
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if status == 200 {
		dic, err := helper.DeserializeJSON[map[string]any](*response)
		if err != nil {
			return nil, err
		}
		list := []T{}
		if values, ok := (*dic)["value"].([]any); ok {
			var requestObject T
			for _, value := range values {
				if valueDic, ok := value.(map[string]any); ok {
					if _, ok := valueDic["@search.text"].(string); ok {
						payload := valueDic
						o, err := requestObject.NewObject(payload)
						if err != nil {
							//purposely skip it since it's for suggest only
							continue
						}
						list = append(list, o.(T))
					}
				}
			}
		}
		return list, nil
	}
	if err != nil {
		return nil, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, err
	}
	return nil, nil
}

// max query length is 50
func (repository *Repository[T]) Autocomplete(ctx context.Context, query string, topK int, fuzzy bool, suggesterName string, searchFields []string, filter string) ([]string, error) {
	if len(query) > 50 {
		return nil, nil
	}
	query, shouldNotSearch := repository.preprocessQuery(query)
	if shouldNotSearch {
		return nil, nil
	}
	url := fmt.Sprintf("%v/indexes('%v')/docs/search.post.autocomplete?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"search":           query,
		"suggesterName":    suggesterName,
		"autocompleteMode": "oneTermWithContext",
		"fuzzy":            fuzzy,
		"highlightPreTag":  "<em>",
		"highlightPostTag": "</em>",
		"minimumCoverage":  80,
		"searchFields":     strings.Join(searchFields, ","),
		"top":              topK,
	}
	if filter != "" {
		body["filter"] = filter
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if status == 200 {
		dic, err := helper.DeserializeJSON[map[string]any](*response)
		if err != nil {
			return nil, err
		}
		list := []string{}
		if values, ok := (*dic)["value"].([]any); ok {
			for _, value := range values {
				if valueDic, ok := value.(map[string]any); ok {
					if searchText, ok := valueDic["queryPlusText"].(string); ok {
						list = append(list, searchText)
					}
				}
			}
		}
		helper.Sort(list, func(a string, b string) int {
			if len(a) < len(b) {
				return -1
			} else if len(a) > len(b) {
				return 1
			}
			return 0
		})
		return list, nil
	}
	if err != nil {
		return nil, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, err
	}
	return nil, nil
}

func (repository *Repository[T]) SearchSemantic(ctx context.Context, query string, embedding []float32, searchVectorFields []string, searchVectorFieldWeights []float32, returnEmbedding bool, filter string, skip int, limit int) ([]T, int, error) {
	query, shouldNotSearch := repository.preprocessQuery(query)
	if shouldNotSearch {
		return nil, 0, nil
	}
	var vectorQueries []map[string]any
	for _, field := range searchVectorFields {
		vectorQuery := map[string]any{
			"fields": field,
			"k":      50,
			"kind":   "vector",
			"vector": embedding,
			//"weight": searchVectorFieldWeights[i],
			//"exhaustive": true,
		}
		vectorQueries = append(vectorQueries, vectorQuery)
	}
	var requestObject T
	url := fmt.Sprintf("%v/indexes/%v/docs/search?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"search":           query,
		"vectorQueries":    vectorQueries,
		"select":           strings.Join(requestObject.ReturnFields(returnEmbedding), ", "),
		"queryType":        "semantic",
		"answers":          "none",
		"captions":         "none",
		"skip":             skip,
		"top":              limit,
		"count":            true,
		"vectorFilterMode": "preFilter",
		//"queryLanguage": "en-US",
	}
	if repository.semanticConfigurationName != "" {
		body["semanticConfiguration"] = repository.semanticConfigurationName
	}
	if filter != "" {
		body["filter"] = filter
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return nil, 0, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, 0, err
	}
	result, err := helper.DeserializeJSON[map[string]any](*response)
	if err != nil {
		return nil, 0, err
	}
	values := (*result)["value"].([]any)
	points := []T{}
	for _, value := range values {
		payload := value.(map[string]any)
		if payload["@search.rerankerScore"] != nil {
			s := float32(payload["@search.rerankerScore"].(float64))
			payload["score"] = s
		} else if payload["@search.score"] != nil {
			s := float32(payload["@search.score"].(float64))
			payload["score"] = s
		}
		o, err := requestObject.NewObject(payload)
		if err != nil {
			return nil, 0, err
		}
		points = append(points, o.(T))
	}
	totalCount := 0
	if c, ok := (*result)["@odata.count"].(float64); ok {
		totalCount = int(c)
	}
	return points, totalCount, nil
}

func (repository *Repository[T]) SearchVector(ctx context.Context, embedding []float32, searchVectorFields []string, searchVectorFieldWeights []float32, returnEmbedding bool, filter string, limit int, threshold float32) ([]T, error) {
	var vectorQueries []map[string]any
	for i, field := range searchVectorFields {
		vectorQuery := map[string]any{
			"fields":     field,
			"k":          limit * 3,
			"kind":       "vector",
			"vector":     embedding,
			"exhaustive": true,
			"weight":     searchVectorFieldWeights[i],
			"threshold": map[string]any{
				"kind":  "searchScore",
				"value": threshold,
			},
		}
		vectorQueries = append(vectorQueries, vectorQuery)
	}
	var requestObject T
	url := fmt.Sprintf("%v/indexes/%v/docs/search?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"vectorQueries": vectorQueries,
		"select":        strings.Join(requestObject.ReturnFields(returnEmbedding), ", "),
		"answers":       "none",
		"captions":      "none",
		"top":           limit,
		"count":         true,
	}
	if filter != "" {
		body["filter"] = filter
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return nil, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, err
	}
	result, err := helper.DeserializeJSON[map[string]any](*response)
	if err != nil {
		return nil, err
	}
	values := (*result)["value"].([]any)
	points := []T{}
	for _, value := range values {
		payload := value.(map[string]any)
		payload["score"] = float32(payload["@search.score"].(float64))
		o, err := requestObject.NewObject(payload)
		if err != nil {
			return nil, err
		}
		points = append(points, o.(T))
	}
	return points, nil
}

// not vector based
func (repository *Repository[T]) SearchText(ctx context.Context, query string, semantic bool, returnEmbedding bool, filter string, orderBy string, desc bool, skip int, limit int) ([]T, int, error) {
	query, shouldNotSearch := repository.preprocessQuery(query)
	if shouldNotSearch {
		return nil, 0, nil
	}
	var requestObject T
	url := fmt.Sprintf("%v/indexes/%v/docs/search?api-version=%v", repository.baseUrl, repository.indexName, repository.apiVersion)
	body := map[string]any{
		"search":   query,
		"select":   strings.Join(requestObject.ReturnFields(returnEmbedding), ", "),
		"answers":  "none",
		"captions": "none",
		"top":      limit,
		"count":    true,
		"skip":     skip,
	}
	if semantic {
		body["queryType"] = "semantic"
		if repository.semanticConfigurationName != "" {
			body["semanticConfiguration"] = repository.semanticConfigurationName
		}
	}
	if filter != "" {
		body["filter"] = filter
	}
	if !semantic && orderBy != "" {
		var order string
		if orderBy != "" {
			order = orderBy
			if desc {
				order += " desc"
			}
		} else {
			order = "entry_date desc"
		}
		body["orderby"] = order
	}
	status, response, _, err := helper.RequestHTTP(ctx, url, "POST", &repository.header, &body)
	if err != nil {
		return nil, 0, err
	}
	if err := repository.handleError(response, status); err != nil {
		return nil, 0, err
	}
	result, err := helper.DeserializeJSON[map[string]any](*response)
	if err != nil {
		return nil, 0, err
	}
	values := (*result)["value"].([]any)
	points := []T{}
	totalCount := int((*result)["@odata.count"].(float64))
	for _, value := range values {
		payload := value.(map[string]any)
		if rerankScore, ok := payload["@search.rerankerScore"].(float64); ok {
			payload["score"] = float32(rerankScore)
		} else if score, ok := payload["@search.score"].(float64); ok {
			payload["score"] = float32(score)
		}
		o, err := requestObject.NewObject(payload)
		if err != nil {
			return nil, 0, err
		}
		points = append(points, o.(T))
	}
	return points, totalCount, nil
}

// utilities
func (repository *Repository[T]) handleError(response *string, status int) error {
	if response == nil {
		return errors.New("no response from azure search")
	}
	dic, err := helper.DeserializeJSON[map[string]any](*response)
	if err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	if (*dic)["error"] != nil {
		errorDic := (*dic)["error"].(map[string]any)
		message := errorDic["message"].(string)
		return fmt.Errorf("status: %d, error: %s", status, message)
	}
	return nil
}
