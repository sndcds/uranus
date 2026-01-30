package sql_utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/sndcds/uranus/app"
)

// BuildSanitizedIlikeCondition sanitizes a search string and appends an ILIKE condition
// and corresponding argument to the provided slices.
//
// Parameters:
//   - inputStr: The raw input string (e.g., from query param).
//   - columnExpr: SQL expression (e.g., "e.title" or "v.city").
//   - label: A descriptive name for error messages (e.g., "title", "city").
//   - startIndex: The index to start SQL placeholders at ($1, $2, ...).
//   - conditions: Pointer to slice of SQL conditions.
//   - args: Pointer to slice of SQL arguments.
//
// Returns:
//   - newArgIndex: The next placeholder index after appending.
//   - error: If sanitization fails.
func BuildSanitizedIlikeCondition(
	inputStr, columnExpr, label string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if inputStr == "" {
		return startIndex, nil
	}

	sanitizedStr, err := SanitizeSearchPattern(inputStr)
	if err != nil {
		return startIndex, fmt.Errorf("%s format error: %s", label, inputStr)
	}

	*conditions = append(*conditions, fmt.Sprintf("%s ILIKE $%d", columnExpr, startIndex))
	*args = append(*args, sanitizedStr)
	return startIndex + 1, nil
}

func BuildSanitizedSearchCondition(
	inputStr string, // the search string
	columnExpr string, // e.g., "e.document_normalized"
	label string, // label for errors
	startIndex int, // starting parameter index ($1, $2, ...)
	conditions *[]string, // append SQL conditions
	args *[]interface{}, // append args
) (int, error) {
	if inputStr == "" {
		return startIndex, nil
	}

	// sanitize input (trim, remove dangerous characters, etc.)
	sanitizedStr, err := SanitizeSearchPattern(inputStr)
	if err != nil {
		return startIndex, fmt.Errorf("%s format error: %s", label, inputStr)
	}

	// Normalize German input so "ä" -> "ae", "ß" -> "ss", etc.
	normalizedInput := normalizeGerman(sanitizedStr)

	// Use ILIKE for substring match with pg_trgm index
	*conditions = append(*conditions, fmt.Sprintf("%s ILIKE '%%' || $%d || '%%'", columnExpr, startIndex))
	*args = append(*args, normalizedInput)

	return startIndex + 1, nil
}

// Example helper to normalize German input in Go
func normalizeGerman(s string) string {
	s = strings.ToLower(s)
	replacements := []struct{ old, new string }{
		{"ä", "ae"},
		{"ö", "oe"},
		{"ü", "ue"},
		{"ß", "ss"},
	}
	for _, r := range replacements {
		s = strings.ReplaceAll(s, r.old, r.new)
	}
	return s
}

// BuildBitmaskCondition constructs a SQL bitmask filter from a string of flags.
//
// It supports both a single integer value (e.g., "16") representing a full bitmask,
// and a comma-separated list of flag indices (e.g., "1,3,5"), which are combined
// into a single uint64 bitmask using bitwise OR (i.e., 1<<flag).
//
// The resulting condition added to `conditions` has the form:
//
//	(columnExpr & $N) = $N
//
// where $N is the placeholder for the bitmask value.
//
// This format is commonly used to match rows where all specified flags are present
// in the column's stored bitmask value.
//
// Parameters:
//   - inputStr: A string representing either a full bitmask (e.g., "16") or individual flag indices (e.g., "1,3,5").
//   - columnExpr: The SQL expression to test the bitmask against (e.g., "ed.accessibility_flags").
//   - label: A human-readable label for use in error messages (e.g., "accessibility_flags").
//   - startIndex: The current index for SQL placeholder numbering (e.g., 1, 2, 3, ...).
//   - conditions: A pointer to a slice of strings where the SQL condition will be appended.
//   - args: A pointer to a slice of interface{} where the corresponding SQL arguments will be appended.
//
// Returns:
//   - The next available placeholder index after inserting the bitmask value.
//   - An error if any flag is invalid or out of the 0–63 range.
//
// Example:
//
//	inputStr = "1,3,5" → bitmask = 1<<1 | 1<<3 | 1<<5 = 42
//	columnExpr = "ed.accessibility_flags"
//	Resulting SQL condition: "(ed.accessibility_flags & $1) = $1"
//	args = [42]
//
// If inputStr is empty, the function does nothing and returns startIndex unchanged.
func BuildBitmaskCondition(
	inputStr, columnExpr, label string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if inputStr == "" {
		return startIndex, nil
	}

	parts := strings.Split(inputStr, ",")
	var bitmask uint64

	for _, part := range parts {
		flagStr := strings.TrimSpace(part)
		flagInt, err := strconv.Atoi(flagStr)
		if err != nil {
			return startIndex, fmt.Errorf("%s format error: %s", label, flagStr)
		}
		if flagInt < 0 || flagInt > 62 {
			return startIndex, fmt.Errorf("%s contains out-of-range flag: %d", label, flagInt)
		}
		bitmask |= 1 << flagInt
	}

	// Add condition and bitmask argument
	*conditions = append(*conditions,
		fmt.Sprintf("(%s & $%d) = $%d", columnExpr, startIndex, startIndex))
	*args = append(*args, bitmask)

	return startIndex + 1, nil
}

// BuildIntRangeCondition parses min and max numeric string inputs and appends a SQL condition
// and arguments to filter a column by a numeric range (e.g., age BETWEEN min AND max).
//
// Parameters:
//   - minStr: Minimum value as a string (e.g., "18").
//   - maxStr: Maximum value as a string (e.g., "65").
//   - column: The column name for the SQL condition (e.g., "age").
//   - startIndex: Starting index for SQL placeholders ($1, $2, ...).
//   - conditions: Pointer to a slice of WHERE clause conditions to be appended to.
//   - args: Pointer to a slice of arguments to be appended to.
//
// Returns:
//   - newArgIndex: The updated argument index after adding new placeholders.
//   - err: An error if parsing of min/max values fails.
func BuildIntRangeCondition(
	minStr, maxStr, column string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if minStr == "" && maxStr == "" {
		return startIndex, nil
	}

	if minStr != "" && maxStr != "" {
		minVal, err := strconv.Atoi(strings.TrimSpace(minStr))
		if err != nil {
			return startIndex, fmt.Errorf("invalid min value: %s", minStr)
		}
		maxVal, err := strconv.Atoi(strings.TrimSpace(maxStr))
		if err != nil {
			return startIndex, fmt.Errorf("invalid max value: %s", maxStr)
		}
		*conditions = append(*conditions, fmt.Sprintf("(%s BETWEEN $%d AND $%d)", column, startIndex, startIndex+1))
		*args = append(*args, minVal, maxVal)
		return startIndex + 2, nil
	}

	if minStr != "" {
		minVal, err := strconv.Atoi(strings.TrimSpace(minStr))
		if err != nil {
			return startIndex, fmt.Errorf("invalid min value: %s", minStr)
		}
		*conditions = append(*conditions, fmt.Sprintf("(%s >= $%d)", column, startIndex))
		*args = append(*args, minVal)
		return startIndex + 1, nil
	}

	// Only maxStr is present
	maxVal, err := strconv.Atoi(strings.TrimSpace(maxStr))
	if err != nil {
		return startIndex, fmt.Errorf("invalid max value: %s", maxStr)
	}
	*conditions = append(*conditions, fmt.Sprintf("(%s <= $%d)", column, startIndex))
	*args = append(*args, maxVal)
	return startIndex + 1, nil
}

// BuildContainedInColumnIntRangeCondition builds a SQL condition for a min/max range.
// - One value: "v BETWEEN COALESCE(minCol,0) AND COALESCE(maxCol,1000)"
// - Two values: "COALESCE(minCol,0) <= v1 AND COALESCE(maxCol,1000) >= v2"
func BuildContainedInColumnIntRangeCondition(
	inputStr, minCol, maxCol string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {

	inputStr = strings.TrimSpace(inputStr)
	if inputStr == "" {
		return startIndex, nil
	}

	parts := strings.Split(inputStr, ",")
	if len(parts) > 2 {
		return startIndex, fmt.Errorf("invalid input: expected one or two integers, got: %s", inputStr)
	}

	if len(parts) == 1 {
		// Single value: value BETWEEN minCol AND maxCol
		val, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return startIndex, fmt.Errorf("invalid integer value: %s", parts[0])
		}
		condition := fmt.Sprintf("($%d BETWEEN %s AND %s)", startIndex, minCol, maxCol)
		*conditions = append(*conditions, condition)
		*args = append(*args, val)
		startIndex++
	} else if len(parts) == 2 {
		// Two values: first >= minCol AND second <= maxCol
		val1, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return startIndex, fmt.Errorf("invalid integer value: %s", parts[0])
		}
		val2, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return startIndex, fmt.Errorf("invalid integer value: %s", parts[1])
		}
		condition := fmt.Sprintf("(%s <= $%d AND %s >= $%d)", minCol, startIndex, maxCol, startIndex+1)
		*conditions = append(*conditions, condition)
		*args = append(*args, val1, val2)
		startIndex += 2
	}

	return startIndex, nil
}

// BuildColumnInIntCondition builds a SQL condition with = ANY($N) for integers.
// idsInput can be either a string ("1,2,3") or a []int.
// expr is the SQL column/expression (e.g., "event_id").
// startIndex is the placeholder number ($1, $2, ...).
// conditions is a pointer to the slice of WHERE conditions to append to.
// args is a pointer to the slice of query arguments to append to.
func BuildColumnInIntCondition(
	idsInput interface{},
	expr string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	var ids []int

	// Parse input
	switch v := idsInput.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return startIndex, nil
		}
		parts := strings.Split(v, ",")
		for _, raw := range parts {
			id, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil {
				return startIndex, fmt.Errorf("invalid integer in input: %s", raw)
			}
			ids = append(ids, id)
		}
	default:
		return startIndex, fmt.Errorf("unsupported input type: %T", idsInput)
	}

	// Build condition
	condition := fmt.Sprintf("(%s = ANY(string_to_array($%d, ',')::int[]))", expr, startIndex)
	*conditions = append(*conditions, condition)

	*args = append(*args, app.IntSliceToCsv(ids))

	return startIndex + 1, nil
}

// BuildInConditionForStringSlice builds an SQL IN/ANY condition for a comma-separated string list
// and adds it as a single array argument for SQL sql.
//
// Parameters:
//   - inputStr: A comma-separated string (e.g., "US, IN, CA").
//   - format: The format string for the SQL condition (e.g., "v.country = ANY(%s)").
//   - label: The label for the input parameter (not used in query, optional).
//   - startIndex: The starting index for SQL placeholders ($1, $2, ...).
//   - conditions: Pointer to slice of SQL conditions to append to.
//   - args: Pointer to slice of query arguments to append to.
//
// Returns:
//   - Updated argument index (incremented by 1 since we pass a single array arg).
//   - error if parsing fails.
func BuildInConditionForStringSlice(inputStr, format, label string, startIndex int, conditions *[]string, args *[]interface{}) (int, error) {
	if inputStr == "" {
		return startIndex, nil
	}

	// Split and trim
	parts := strings.Split(inputStr, ",")
	var values []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}

	if len(values) == 0 {
		return startIndex, nil
	}

	// Create a single placeholder for the array argument
	placeholder := fmt.Sprintf("$%d", startIndex)

	// Format the condition string with the placeholder
	condition := fmt.Sprintf(format, placeholder)
	*conditions = append(*conditions, condition)

	// Append the slice as one array argument
	*args = append(*args, values)

	// Return next argument index (we used one)
	return startIndex + 1, nil
}

// BuildTimeCondition parses a flexible time range string and appends a SQL BETWEEN condition
// to the provided conditions and args slices. It returns the updated argument index.
//
// Parameters:
//   - timeStr: A comma-separated string with two times (e.g., "9,1730").
//   - field: The SQL field to apply the condition to.
//   - label: A human-readable label for error reporting.
//   - argIndex: The starting index for SQL parameter placeholders ($N).
//   - conditions: Pointer to the slice of condition strings to append to.
//   - args: Pointer to the slice of SQL arguments to append to.
//
// Returns:
//   - newArgIndex: The next available argument index after appending.
//   - err: An error if input is invalid or cannot be parsed.
func BuildTimeCondition(
	timeStr, field, label string,
	argIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if timeStr == "" {
		return argIndex, nil
	}

	parts := strings.Split(timeStr, ",")
	if len(parts) != 2 {
		return argIndex, fmt.Errorf("%s must have exactly two times separated by a comma: %s", label, timeStr)
	}

	parseFlexibleTime := func(s string) (string, error) {
		s = strings.TrimSpace(s)
		switch len(s) {
		case 1, 2:
			hour, err := strconv.Atoi(s)
			if err != nil || hour < 0 || hour > 23 {
				return "", fmt.Errorf("invalid hour: %s", s)
			}
			return fmt.Sprintf("%02d:00", hour), nil
		case 4:
			hour, err1 := strconv.Atoi(s[:2])
			min, err2 := strconv.Atoi(s[2:])
			if err1 != nil || err2 != nil || hour < 0 || hour > 23 || min < 0 || min > 59 {
				return "", fmt.Errorf("invalid HHMM format: %s", s)
			}
			return fmt.Sprintf("%02d:%02d", hour, min), nil
		default:
			return "", fmt.Errorf("invalid time format: %s", s)
		}
	}

	startStr, err := parseFlexibleTime(parts[0])
	if err != nil {
		return argIndex, fmt.Errorf("%s start time is invalid: %v", label, err)
	}
	endStr, err := parseFlexibleTime(parts[1])
	if err != nil {
		return argIndex, fmt.Errorf("%s end time is invalid: %v", label, err)
	}

	condition := fmt.Sprintf("TO_CHAR(%s, 'HH24:MI') BETWEEN $%d AND $%d", field, argIndex, argIndex+1)
	*conditions = append(*conditions, condition)
	*args = append(*args, startStr, endStr)

	return argIndex + 2, nil
}

// BuildLikeConditions parses a comma-separated string of search patterns and appends an ILIKE
// condition to the provided slices for use in SQL WHERE clauses.
//
// Parameters:
//   - inputStr: A comma-separated list of search terms (e.g., "art,design").
//   - field: The SQL column to apply the ILIKE condition to (e.g., "title").
//   - argIndex: The starting index for SQL placeholders ($1, $2, etc.).
//   - conditions: A pointer to a slice of condition strings (modified in place).
//   - args: A pointer to a slice of SQL argument values (modified in place).
//
// Behavior:
//   - Wildcard `*` in inputStr will be converted to SQL wildcard `%`.
//   - Each value becomes a `field ILIKE $N` clause, joined by `OR`.
//   - The final condition is wrapped in parentheses: e.g., `(title ILIKE $1 OR title ILIKE $2)`.
//
// Returns:
//   - The next available SQL argument index after all patterns.
//   - An error if input is malformed (currently always nil).
func BuildLikeConditions(inputStr, field string, argIndex int, conditions *[]string, args *[]interface{}) (int, error) {
	if inputStr == "" {
		return argIndex, nil
	}

	parts := strings.Split(inputStr, ",")
	var likeClauses []string

	for _, part := range parts {
		pattern := strings.TrimSpace(part)
		pattern = strings.ReplaceAll(pattern, "*", "%")
		likeClauses = append(likeClauses, fmt.Sprintf("%s ILIKE $%d", field, argIndex))
		*args = append(*args, pattern)
		argIndex++
	}

	// Combine all ILIKE clauses with OR
	condition := "(" + strings.Join(likeClauses, " OR ") + ")"
	*conditions = append(*conditions, condition)

	return argIndex, nil
}

// SanitizeSearchPattern sanitizes a search pattern by handling wildcard characters and ensuring
// that the pattern is safe for use in SQL sql.
//
// It processes the input string to:
//   - Escape literal asterisks (`\*`), replacing them with a placeholder before converting.
//   - Replace unescaped asterisks (`*`) with the SQL wildcard `%`.
//   - Restore literal asterisks from the placeholder back to `*`.
//   - Reject patterns with too many wildcards (e.g., "%%") or leading wildcards.
//
// Parameters:
//   - input: The search pattern string that needs to be sanitized. This can include asterisks (`*`),
//     which will be treated as SQL wildcards. If empty, it returns an empty string without error.
//
// Returns:
//   - A sanitized version of the input string, where unsafe or invalid patterns are modified or rejected.
//   - An error if the sanitized pattern contains too many wildcards or any other dangerous patterns.
//
// Errors:
//   - If the pattern contains too many `%` symbols (e.g., `%%` or patterns with leading wildcards),
//     an error will be returned with the message "too many wildcards in city filter".
//
// Example Usage:
//
//	sanitizedPattern, err := SanitizeSearchPattern("New*York")
//	if err != nil {
//	  // handle error
//	}
func SanitizeSearchPattern(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	// Handle escaped asterisks: \* becomes literal *
	re := regexp.MustCompile(`\\\*`)
	safeInput := re.ReplaceAllString(input, "[ASTERISK_PLACEHOLDER]")

	// Replace unescaped * with SQL wildcard %
	safeInput = strings.ReplaceAll(safeInput, "*", "%")

	// Restore literal asterisks
	safeInput = strings.ReplaceAll(safeInput, "[ASTERISK_PLACEHOLDER]", "*")

	// Optional: Reject dangerous patterns like "%%" or leading wildcards
	if strings.Count(safeInput, "%") > 2 {
		return "", fmt.Errorf("too many wildcards in city filter")
	}

	return safeInput, nil
}

// BuildLimitOffsetClause parses limit and offset strings into integers,
// validates them, and appends the appropriate LIMIT and OFFSET clauses
// and arguments to the provided args slice.
//
// Parameters:
//   - limitStr:   Query string for limit (may be empty).
//   - offsetStr:  Query string for offset (may be empty).
//   - startIndex: Index for SQL placeholders ($N).
//   - args:       Pointer to the slice of arguments to be extended.
//
// Returns:
//   - clause:     A SQL clause like "LIMIT $1 OFFSET $2", or just one part.
//   - newIndex:   The updated index after adding args.
//   - err:        A validation error, if any.
func BuildLimitOffsetClause(limitStr, offsetStr string, startIndex int, args *[]interface{}) (string, int, error) {
	var clauses []string
	argIndex := startIndex

	// Parse limit
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return "", argIndex, fmt.Errorf("invalid limit: %s", limitStr)
		}
		clauses = append(clauses, fmt.Sprintf("LIMIT $%d", argIndex))
		*args = append(*args, limit)
		argIndex++
	}

	// Parse offset
	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return "", argIndex, fmt.Errorf("invalid offset: %s", offsetStr)
		}
		clauses = append(clauses, fmt.Sprintf("OFFSET $%d", argIndex))
		*args = append(*args, offset)
		argIndex++
	}

	return strings.Join(clauses, " "), argIndex, nil
}

// BuildGeoRadiusCondition adds a PostGIS ST_DWithin condition to the given conditions slice,
// if all three input strings (lonStr, latStr, radiusStr) are valid and non-empty.
//
// Parameters:
//   - lonStr, latStr, radiusStr: string inputs for longitude, latitude, and radius in meters.
//   - columnExpr: the SQL expression for the geometry column (e.g., "v.wkb_pos").
//   - startIndex: the starting placeholder index for the SQL arguments.
//   - conditions: a pointer to the slice of WHERE conditions to append to.
//   - args: a pointer to the slice of SQL arguments.
//
// Returns:
//   - nextArgIndex: the next available argument index after this condition.
//   - err: any error encountered while parsing the inputs.
func BuildGeoRadiusCondition(
	lonStr, latStr, radiusStr, columnExpr string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if lonStr == "" || latStr == "" || radiusStr == "" {
		return startIndex, nil
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return startIndex, fmt.Errorf("lon '%s' is invalid", lonStr)
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return startIndex, fmt.Errorf("lat '%s' is invalid", latStr)
	}
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		return startIndex, fmt.Errorf("radius '%s' is invalid", radiusStr)
	}

	*conditions = append(*conditions,
		fmt.Sprintf(
			"ST_DWithin(%s::geography, ST_MakePoint($%d, $%d)::geography, $%d)",
			columnExpr, startIndex, startIndex+1, startIndex+2,
		),
	)
	*args = append(*args, lon, lat, radius)
	return startIndex + 3, nil
}

func BuildJSONArrayIntCondition(
	input string,
	jsonbColumn string, // e.g. "ep.types"
	jsonIndex int, // 0 = type_id, 1 = genre_id
	argIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {

	if input == "" {
		return argIndex, nil
	}

	ids, err := app.ParseIntSliceCsv(input)
	if err != nil {
		return argIndex, err
	}

	condition := fmt.Sprintf(
		`EXISTS (SELECT 1 FROM jsonb_array_elements(%s) AS t(elem) WHERE (elem->>%d)::int = ANY($%d))`,
		jsonbColumn, jsonIndex, argIndex)
	*conditions = append(*conditions, condition)

	*args = append(*args, ids)
	return argIndex + 1, nil
}

// BuildArrayContainsCondition builds a SQL condition for a text[] column
// to check if it contains all of the provided comma-separated values.
// Example: tags=young,festival
func BuildArrayContainsCondition(
	csv string, // comma-separated values
	column string, // e.g. "tags"
	argIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	values := strings.Split(csv, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	if len(values) == 0 {
		return argIndex, nil
	}

	// add the argument as text[] for @> operator
	*args = append(*args, pq.Array(values))
	*conditions = append(*conditions, fmt.Sprintf("%s @> $%d::text[]", column, argIndex))
	argIndex++
	return argIndex, nil
}
