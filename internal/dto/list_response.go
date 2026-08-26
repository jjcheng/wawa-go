package dto

type ListResponse[T any] struct {
	Items          []T             `json:"items"`
	NumberOfPages  int             `json:"number_of_pages"`
	NumberOfItems  int             `json:"number_of_items"`
	NextPageOffset any             `json:"next_page_offset,omitempty"`
	AdditionalData *map[string]any `json:"additional_data,omitempty"`
}

func NewPagedListResponse[T any](items []T, numberOfPages int, numberOfItems int) ListResponse[T] {
	container := ListResponse[T]{
		Items:         items,
		NumberOfPages: numberOfPages,
		NumberOfItems: numberOfItems,
	}
	if len(container.Items) == 0 {
		container.Items = []T{}
	}
	return container
}

func NewOffsetListResponse[T any](items []T, nextPageOffset any, additionalData *map[string]any) ListResponse[T] {
	container := ListResponse[T]{
		Items:          items,
		NextPageOffset: nextPageOffset,
		AdditionalData: additionalData,
	}
	if len(container.Items) == 0 {
		container.Items = []T{}
	}
	return container
}
