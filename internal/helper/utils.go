package helper

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/dustin/go-humanize"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/google/uuid"
	"github.com/twpayne/go-polyline"
	"mvdan.cc/xurls/v2"
)

func DefaultIfEmpty(val string, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func ConcatMultipleSlices[T any](slices [][]T) []T {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]T, totalLen)
	var i int
	for _, s := range slices {
		i += copy(result[i:], s)
	}
	return result
}

func GetDateTimeString(dateTime time.Time, local bool) string {
	if local {
		loc, err := time.LoadLocation("Asia/Singapore")
		if err != nil {
			return dateTime.Format("Monday, 2006-01-02 15:04:05")
		}
		t := time.Now().In(loc)
		return t.Format("Monday, 2006-01-02 15:04:05")
	}
	return dateTime.Format("Monday, 2006-01-02 15:04:05")
}

func GetDateString(dateTime time.Time, local bool) string {
	if local {
		loc, err := time.LoadLocation("Asia/Singapore")
		if err != nil {
			return dateTime.Format("Monday, 2006-01-02")
		}
		t := time.Now().In(loc)
		return t.Format("Monday, 2006-01-02")
	}
	return dateTime.Format("Monday, 2006-01-02")
}

func GetTimeString(dateTime time.Time, local bool) string {
	if local {
		loc, err := time.LoadLocation("Asia/Singapore")
		if err != nil {
			return dateTime.Format("15:04")
		}
		t := time.Now().In(loc)
		return t.Format("15:04")
	}
	return dateTime.Format("15:04")
}

func Map[T any, V any](list []T, fn func(T) V) []V {
	result := make([]V, len(list))
	for i, t := range list {
		result[i] = fn(t)
	}
	return result
}

// this function returns a newly created list
func Filter[T any](list []T, fn func(T) bool) []T {
	result := []T{}
	if len(list) == 0 {
		return result
	}
	for i := 0; i <= len(list)-1; i++ {
		if fn(list[i]) {
			result = append(result, list[i])
		}
	}
	return result
}

// this function does not create new list
func FilterPointers[T any](list []T, fn func(T) bool) []*T {
	result := []*T{}
	if len(list) == 0 {
		return result
	}
	for i := 0; i <= len(list)-1; i++ {
		if fn(list[i]) {
			result = append(result, &list[i])
		}
	}
	return result
}

func First[T any](list []T, fn func(T) bool) *T {
	if len(list) == 0 {
		return nil
	}
	for i := 0; i <= len(list)-1; i++ {
		if fn(list[i]) {
			return &list[i]
		}
	}
	return nil
}

// Min returns a pointer to the element with the minimum value based on the selector function
// Returns nil if the list is empty
func Min[T any, C interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string
}](list []T, fn func(T) C) *T {
	if len(list) == 0 {
		return nil
	}
	minIndex := 0
	minValue := fn(list[0])
	for i := 1; i < len(list); i++ {
		value := fn(list[i])
		if value < minValue {
			minValue = value
			minIndex = i
		}
	}
	return &list[minIndex]
}

// Max returns a pointer to the element with the maximum value based on the selector function
// Returns nil if the list is empty
func Max[T any, C interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string
}](list []T, fn func(T) C) *T {
	if len(list) == 0 {
		return nil
	}
	maxIndex := 0
	maxValue := fn(list[0])
	for i := 1; i < len(list); i++ {
		value := fn(list[i])
		if value > maxValue {
			maxValue = value
			maxIndex = i
		}
	}
	return &list[maxIndex]
}

func Last[T any](list []T, fn func(T) bool) *T {
	if len(list) == 0 {
		return nil
	}
	for i := len(list) - 1; i >= 0; i-- {
		if fn(list[i]) {
			return &list[i]
		}
	}
	return nil
}

func Any[T any](list []T, fn func(T) bool) bool {
	return slices.ContainsFunc(list, fn)
}

func Reduce[T, U any](slice []T, initialValue U, reducer func(U, T) U) U {
	result := initialValue
	for _, value := range slice {
		result = reducer(result, value)
	}
	return result
}

func Count[T any](slice []T, fn func(T) bool) int {
	result := 0
	for _, value := range slice {
		if fn(value) {
			result += 1
		}
	}
	return result
}

func Group[T any, K any](list []T, fn func(T) K) map[any][]T {
	m := make(map[any][]T)
	for _, item := range list {
		key := fn(item)
		if value, ok := m[key]; ok {
			m[key] = append(value, item)
		} else {
			m[key] = []T{item}
		}
	}
	return m
}

func Distinct[T comparable](list []T) []T {
	if len(list) <= 1 {
		return list
	}
	distincts := []T{}
	for _, item := range list {
		if !Any(distincts, func(i T) bool {
			return i == item
		}) {
			distincts = append(distincts, item)
		}
	}
	return distincts
}

func Take[T any](list []T, length int) []T {
	if len(list) < length {
		return list
	}
	return list[0 : length+1]
}

func Select[T any, TResult any](list []T, fn func(T) TResult) []TResult {
	l := []TResult{}
	for _, item := range list {
		l = append(l, fn(item))
	}
	return l
}

func Sort[T any](list []T, fn func(a T, b T) int) {
	slices.SortFunc(list, func(a, b T) int {
		return fn(a, b)
	})
}

func RoundTo2Decimals(val float64) float64 {
	return math.Round(val*100) / 100
}

func ToThousands[T float64 | int64 | int32 | int](number T) string {
	switch v := any(number).(type) {
	case float64:
		return humanize.Commaf(v)
	case int64:
		return humanize.Comma(v)
	case int:
		return humanize.Comma(int64(v))
	case int32:
		return humanize.Comma(int64(v))
	default:
		return ""
	}
}

func Shuffle[T any](list []T) []T {
	// Use crypto/rand for cryptographically secure shuffling
	for i := len(list) - 1; i > 0; i-- {
		// Generate secure random index
		randomBytes := make([]byte, 4)
		_, err := rand.Read(randomBytes)
		if err != nil {
			// If crypto/rand fails, this is a serious system issue
			// Return the list unshuffled instead of using weak random
			log.Printf("Warning: crypto/rand failed during shuffle: %v", err)
			return list
		}

		// Convert bytes to int and get random index
		randomInt := int(randomBytes[0])<<24 | int(randomBytes[1])<<16 | int(randomBytes[2])<<8 | int(randomBytes[3])
		if randomInt < 0 {
			randomInt = -randomInt
		}
		j := randomInt % (i + 1)

		// Swap elements
		list[i], list[j] = list[j], list[i]
	}
	return list
}

// GenerateCombinations generates all possible combinations of items from a list
// up to a specified maximum size. Results are ordered by combination size (smallest first).
// maxSize: maximum size of combinations (0 or negative means generate all sizes up to len(items))
// Returns a slice of slices, where each inner slice is a combination.
func GenerateCombinations[T any](items []T, maxSize int) [][]T {
	if len(items) == 0 {
		return [][]T{}
	}

	// If maxSize is not specified or invalid, use the full length
	if maxSize <= 0 || maxSize > len(items) {
		maxSize = len(items)
	}

	var result [][]T

	// Generate combinations for each size from 1 to maxSize
	for size := 1; size <= maxSize; size++ {
		combinations := generateCombinationsOfSize(items, size)
		result = append(result, combinations...)
	}

	return result
}

// generateCombinationsOfSize generates all combinations of a specific size
func generateCombinationsOfSize[T any](items []T, size int) [][]T {
	var result [][]T
	var current []T

	var generate func(start int)
	generate = func(start int) {
		if len(current) == size {
			// Make a copy of current combination
			combination := make([]T, size)
			copy(combination, current)
			result = append(result, combination)
			return
		}

		for i := start; i < len(items); i++ {
			current = append(current, items[i])
			generate(i + 1)
			current = current[:len(current)-1]
		}
	}

	generate(0)
	return result
}

func ReplaceAtIndex(str string, replacement string, index int, length int) string {
	runes := []rune(str)
	replacementRunes := []rune(replacement)
	for rIndex := range runes {
		if rIndex < index {
			continue
		}
		if rIndex-index >= len(replacementRunes) {
			if rIndex-index < length {
				runes[rIndex] = 999999999
			}
		} else {
			runes[rIndex] = replacementRunes[rIndex-index]
		}
	}

	runes = Filter(runes, func(r rune) bool {
		return r != 999999999
	})
	return string(runes)
}

func Remove[T comparable](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

func CalculateCosineSimilarity(a []float32, b []float32) (float32, error) {
	count := 0
	length_a := len(a)
	length_b := len(b)
	if length_a > length_b {
		count = length_a
	} else {
		count = length_b
	}
	var sumA float32 = 0.0
	var s1 float32 = 0.0
	var s2 float32 = 0.0
	for k := 0; k < count; k++ {
		if k >= length_a {
			s2 += float32(math.Pow(float64(b[k]), 2))
			continue
		}
		if k >= length_b {
			s1 += float32(math.Pow(float64(a[k]), 2))
			continue
		}
		sumA += a[k] * b[k]
		s1 += float32(math.Pow(float64(a[k]), 2))
		s2 += float32(math.Pow(float64(b[k]), 2))
	}
	if s1 == 0 || s2 == 0 {
		return 0.0, fmt.Errorf("vectors should not be null (all zeros)")
	}
	return float32(sumA) / float32((math.Sqrt(float64(s1)) * math.Sqrt(float64(s2)))), nil
}

func GetNumbersFromText(text string) []float64 {
	re := regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	if !re.MatchString(text) {
		return []float64{}
	}
	submatchall := re.FindAllString(text, -1)
	numbers := []float64{}
	for _, element := range submatchall {
		if s, err := strconv.ParseFloat(element, 64); err == nil {
			numbers = append(numbers, s)
		}
	}
	return numbers
}

func GetTimeStamp(time time.Time) int64 {
	return time.UnixNano() / 1e6
}

func WriteToFile(content string, filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func ReadFromFile(filePath string) (*string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	str := string(b)
	return &str, nil
}

func SantinizeDocumentText(text string) string {
	t := strings.Replace(text, ":unselected: o", "-", -1)
	t = strings.Replace(t, ":selected:", "✓", -1)
	return t
}

func ConvertToMarkdown(text string) (string, error) {
	converter := md.NewConverter("", true, &md.Options{})
	converter.Keep("iframe", "embed", "video", "object", "canvas", "source")
	return converter.ConvertString(text)
}

func ConvertMarkdownToHTML(md string) string {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(markdown.Render(doc, renderer))
}

func IsImageURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

func ReplaceImageUrlsWithImgTag(text string, format string) string {
	urls := ExtractURLs(text)
	format = strings.ToUpper(format)
	result := text
	for _, u := range urls {
		if IsImageURL(u) {
			var replacement string
			switch format {
			case "MARKDOWN":
				replacement = "![](" + u + ")"
			default: // HTML
				replacement = `<img src="` + u + `" />`
			}
			result = strings.ReplaceAll(result, u, replacement)
		}
	}
	return result
}

func Reverse(s any) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func PrintRequestStartEnd(text string, isStart bool, sessionId string, requestTime *time.Time, logMessages *[]string, stream bool) {
	if isStart {
		LogMessage(logMessages, fmt.Sprintf("=== %v ===", text))
		if sessionId != "" {
			LogMessage(logMessages, fmt.Sprintf("session id: %v\n", sessionId))
		}
	} else {
		if requestTime != nil {
			ms := GetTimeStamp(time.Now()) - requestTime.UnixNano()/1e6
			if stream {
				LogMessage(logMessages, fmt.Sprintf("streaming time elapsed: %v\n", ms))
			} else {
				LogMessage(logMessages, fmt.Sprintf("non-streaming time elapsed: %v\n", ms))
			}
		}
		LogMessage(logMessages, fmt.Sprintf("=== %v ===", text))
	}
}

// for most cases, append \n at the end
func LogMessage(logs *[]string, message string) {
	if logs == nil {
		return
	}
	now := time.Now().Format("2006/01/02 15:04:05")
	newMessage := logMessage(message)
	*logs = append(*logs, fmt.Sprintf("%s %s", now, strings.TrimSpace(newMessage)))
}

func logMessage(message string) string {
	log.Print(message)
	return message
}

// between 8 and 20 characters and has atlease 1 letter and 1 number
func ValidatePassword(password string) error {
	hasMin8 := len(password) >= 8 && len(password) <= 20
	hasLetter := false
	hasNumber := false
	for _, char := range password {
		if unicode.IsNumber(char) {
			hasNumber = true
		} else if unicode.IsLetter(char) {
			hasLetter = true
		}
	}
	if hasMin8 && hasLetter && hasNumber {
		return nil
	}
	return errors.New("password must be at between 8 and 20 characters with at least 1 letter and 1 number")
}

func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}

func RemoveUnicode(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, text)
	return text
}

// returns a pointer to the provided value
func ConvertToPointer[T any](v T) *T {
	return &v
}

// returns a slice of *T from the specified values
func ConvertSliceOfPointers[T any](vv ...T) []*T {
	slc := make([]*T, len(vv))
	for i := range vv {
		slc[i] = ConvertToPointer(vv[i])
	}
	return slc
}

func CloneSlice[T any](slice []T) []T {
	copied := make([]T, len(slice))
	copy(copied, slice)
	return copied
}

func CloneMap(dic map[string]any) map[string]any {
	newDic := map[string]any{}
	for k, v := range dic {
		newDic[k] = v
	}
	return newDic
}

func IndexOf[T any](slice []T, fn func(T) bool) *int {
	for i, item := range slice {
		if fn(item) {
			return &i
		}
	}
	return nil
}

func LastIndexOf[T any](slice []T, fn func(T) bool) *int {
	var foundIndex *int
	for i, item := range slice {
		if fn(item) {
			foundIndex = ConvertToPointer(i)
		}
	}
	return foundIndex
}

func GetAllKeys(dic map[string]any) []any {
	keys := make([]any, 0, len(dic))
	for k := range dic {
		keys = append(keys, k)
	}
	return keys
}

func GetTimeElpsedInMS(t time.Time) int64 {
	ms := GetTimeStamp(time.Now()) - t.UnixNano()/1e6
	return ms
}

func GetTimeDifferenceInMS(endAt time.Time, startAt time.Time) int64 {
	return endAt.UnixNano()/1e6 - startAt.UnixNano()/1e6
}

func CalculateDistanceBetween2PointsKM(latitude1 float64, longitude1 float64, latitude2 float64, longitude2 float64) float64 {
	lat1 := degreesToRadians(latitude1)
	lon1 := degreesToRadians(longitude1)
	lat2 := degreesToRadians(latitude2)
	lon2 := degreesToRadians(longitude2)
	dlat := lat2 - lat1
	dlon := lon2 - lon1
	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return 6371 * c
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// returning format: [[latitude, longitude]]
func DecodeGoogleMapPath(encodedPath string) ([][]float64, error) {
	coords, _, err := polyline.DecodeCoords([]byte(encodedPath))
	if err != nil {
		return nil, err
	}
	return coords, nil
}

func GetText(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetStringUntil(s string, until string) string {
	dotIndex := strings.Index(s, until)
	if dotIndex == -1 {
		return s
	}
	return s[:dotIndex]
}

func ExtractContentByTag(input string, startTag string, endTag string) (string, error) {
	startIdx := strings.Index(input, startTag)
	if startIdx == -1 {
		return "", fmt.Errorf("start tag not found")
	}
	endIdx := strings.Index(input, endTag)
	if endIdx == -1 {
		return "", fmt.Errorf("end tag not found")
	}
	contentStart := startIdx + len(startTag)
	content := input[contentStart:endIdx]
	return strings.TrimSpace(content), nil
}

func ExtractJSONCodeBlocks(input string) []string {
	// Regex to match ```json ... ```
	re := regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")
	matches := re.FindAllStringSubmatch(input, -1)
	var results []string
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}
	return results
}

func GetLastCharacter(text string) string {
	chars := []rune(text)
	lastCharacter := string(chars[len(chars)-1])
	return lastCharacter
}

func PadLeft(s string, targetLength int, padChar rune) string {
	padCount := targetLength - len(s)
	if padCount > 0 {
		return strings.Repeat(string(padChar), padCount) + s
	}
	return s
}

func CheckExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false // some other error (e.g. permission denied)
	}
	return info.IsDir()
}

func CheckContainsFileWithExtension(dir string, extension string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip subdirectories if you want only top-level files
			continue
		}
		if filepath.Ext(entry.Name()) == extension {
			return true, nil
		}
	}
	return false, nil
}

func Insert[T any](slice []T, element T, index int) []T {
	if index < 0 || index > len(slice) {
		// Index out of range; return original slice or handle error as needed
		return slice
	}
	// Append a zero value element to extend the slice by 1 and shift elements after index right by 1
	var zero T
	slice = append(slice, zero)
	copy(slice[index+1:], slice[index:])
	slice[index] = element
	return slice
}

func GenerateKey(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func ConvertStringToTime(dateString string) (*time.Time, error) {
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func GetValue(a any, fieldName string) (any, error) {
	v := reflect.ValueOf(a)
	// If it's an interface, unwrap it
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, errors.New("cannot get field from nil interface")
		}
		v = v.Elem()
	}
	// If it's a pointer, dereference it
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, errors.New("cannot get field from nil pointer")
		}
		v = v.Elem()
	}
	// Now ensure we're working with a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct or pointer to struct, got %s", v.Kind())
	}
	// Access the field by name
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("no such field: %s", fieldName)
	}
	// If the field is a pointer, optionally dereference to actual value
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil, fmt.Errorf("field %s is nil pointer", fieldName)
		}
		return field.Elem().Interface(), nil
	}
	return field.Interface(), nil
}

// a must be a pointer
func SetValue(a any, fieldName string, value any) error {
	v := reflect.ValueOf(a)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return errors.New("input must be a non-nil pointer to a struct")
	}

	v = reflect.ValueOf(a).Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct, got pointer to %s", v.Kind())
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("no such field: %s", fieldName)
	}
	if !field.CanSet() {
		return fmt.Errorf("cannot set field %s: unexported or unaddressable", fieldName)
	}

	val := reflect.ValueOf(value)
	if !val.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("provided value type (%s) doesn't match field type (%s)", val.Type(), field.Type())
	}

	field.Set(val)
	return nil
}

func ConvertToArray(s any) []any {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Slice {
		return nil // or panic/error
	}
	result := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result
}

func EncodeToBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func DecodeFromBase64(input string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func RandomUpTo(max int) int {
	// Use crypto/rand for cryptographically secure random numbers
	if max <= 0 {
		return 0
	}

	// Generate random bytes
	randomBytes := make([]byte, 4) // 4 bytes for int32
	_, err := rand.Read(randomBytes)
	if err != nil {
		// If crypto/rand fails, this is a serious system issue
		// Return 0 instead of falling back to weak random
		log.Printf("Warning: crypto/rand failed: %v", err)
		return 0
	}

	// Convert bytes to int and ensure it's within range
	randomInt := int(randomBytes[0])<<24 | int(randomBytes[1])<<16 | int(randomBytes[2])<<8 | int(randomBytes[3])
	if randomInt < 0 {
		randomInt = -randomInt
	}

	return randomInt % max
}

func IsValidPhoneNumber(phone string) bool {
	re := regexp.MustCompile(`^\+?\d{8,15}$`)
	return re.MatchString(phone)
}

func RemoveURLQueryStrings(text string) string {
	// Basic URL regex (http, https)
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	return urlRegex.ReplaceAllStringFunc(text, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil {
			return raw // If parsing fails, keep original
		}
		parsed.RawQuery = "" // Remove query string
		parsed.Fragment = "" // Optional: remove fragment (#section), remove if you want to keep it
		return parsed.String()
	})
}

func ExtractURLs(input string) []string {
	rx := xurls.Strict() // or xurls.Relaxed() for broader matching
	return rx.FindAllString(input, -1)
}

func ExtractCoordinate(input string) (string, string) {
	re := regexp.MustCompile(`(1\.[0-9]+)\D+(103\.[0-9]+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 3 {
		return "", ""
	}
	return matches[1], matches[2]
	// lat, err1 := strconv.ParseFloat(matches[1], 64)
	// lng, err2 := strconv.ParseFloat(matches[2], 64)
	// if err1 != nil || err2 != nil {
	// 	return lat, lng
	// }
	// return 0, 0
}

func IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

func CheckIsImage(resourceURL string) (bool, string) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Head(resourceURL)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		return false, ""
	}
	// Try to get file name from Content-Disposition header
	contentDisposition := resp.Header.Get("Content-Disposition")
	if strings.Contains(contentDisposition, "filename=") {
		parts := strings.Split(contentDisposition, "filename=")
		if len(parts) > 1 {
			// Clean up quotes if present
			fileName := strings.Trim(parts[1], `"`)
			return true, fileName
		}
	}
	// Fallback: extract from URL path
	u, err := url.Parse(resourceURL)
	if err == nil {
		return true, path.Base(u.Path)
	}
	return true, ""
}

func CheckIsValidURL(str string) bool {
	u, err := url.ParseRequestURI(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func CheckIsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func CurrentYear() int {
	return time.Now().Year()
}

func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}
