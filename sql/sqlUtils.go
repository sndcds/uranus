package sql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

// BuildContainedInColumnRangeCondition builds a condition to check if one or two integers
// are within a database range (minCol to maxCol), treating NULLs as defaults.
//
// - One int: "($1 BETWEEN COALESCE(minCol, 0) AND COALESCE(maxCol, 1000))"
// - Two ints: same for each, combined with AND.
func BuildContainedInColumnRangeCondition(
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

	coalescedMin := fmt.Sprintf("COALESCE(%s, 0)", minCol)
	coalescedMax := fmt.Sprintf("COALESCE(%s, 1000)", maxCol)

	for _, part := range parts {
		valStr := strings.TrimSpace(part)
		val, err := strconv.Atoi(valStr)
		if err != nil {
			return startIndex, fmt.Errorf("invalid integer value: %s", valStr)
		}
		*conditions = append(*conditions,
			fmt.Sprintf("($%d BETWEEN %s AND %s)", startIndex, coalescedMin, coalescedMax))
		*args = append(*args, val)
		startIndex++
	}

	return startIndex, nil
}

// BuildColumnInIntCondition adds a SQL WHERE condition for a column to match one or more integer values.
//
// Parameters:
//   - idStr: A comma-separated list of integer values (e.g., "1,2,3").
//   - expr: The SQL column or expression to match against (e.g., "user_id").
//   - label: A human-readable label used in error messages (e.g., "DbUser ID").
//   - startIndex: The current argument index for SQL placeholder numbering.
//   - conditions: A pointer to a slice of SQL condition strings to be appended to.
//   - args: A pointer to a slice of arguments to be passed to the SQL query.
//
// Returns:
//   - newIndex: The updated argument index after processing.
//   - err: An error if parsing fails or idStr contains invalid integers.
func BuildColumnInIntCondition(
	idStr string,
	expr string,
	label string,
	startIndex int,
	conditions *[]string,
	args *[]interface{},
) (int, error) {
	if idStr == "" {
		return startIndex, nil
	}

	ids := strings.Split(idStr, ",")
	placeholders := []string{}

	for i, raw := range ids {
		id := strings.TrimSpace(raw)
		n, err := strconv.Atoi(id)
		if err != nil {
			return startIndex, fmt.Errorf("%s %s is invalid", label, id)
		}
		*args = append(*args, n)
		placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
	}

	var condition string
	if len(placeholders) == 1 {
		condition = fmt.Sprintf("(%s = %s)", expr, placeholders[0])
	} else {
		condition = fmt.Sprintf("(%s IN (%s))", expr, strings.Join(placeholders, ","))
	}

	*conditions = append(*conditions, condition)
	return startIndex + len(placeholders), nil
}

// BuildInConditionForStringSlice builds an SQL IN condition for a comma-separated string list.
// It also handles the transformation of the string list into an argument array for SQL queries.
//
// Parameters:
//   - inputStr: A comma-separated string (e.g., "US, IN, CA") to be used in the IN condition.
//   - format: The format string for the SQL condition (e.g., "v.country_code IN (%s)").
//   - label: The label for the input parameter (e.g., "country_codes").
//   - startIndex: The starting index for SQL placeholders ($1, $2, ...).
//   - conditions: A pointer to the slice of conditions to append to.
//   - args: A pointer to the slice of arguments to append to.
//
// Returns:
//   - The updated argument index after the condition is added.
//   - An error if any parsing issues occur (e.g., invalid formatting).
func BuildInConditionForStringSlice(inputStr, format, label string, startIndex int, conditions *[]string, args *[]interface{}) (int, error) {
	if inputStr == "" {
		return startIndex, nil
	}

	// Split the input string into individual values, trim spaces
	parts := strings.Split(inputStr, ",")
	var values []string
	for _, part := range parts {
		values = append(values, strings.TrimSpace(part))
	}

	// Create placeholders for the SQL IN clause
	var placeholders []string
	for i := range values {
		placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
	}

	// Create the IN condition and append it to the conditions slice
	condition := fmt.Sprintf(format, strings.Join(placeholders, ", "))
	*conditions = append(*conditions, condition)

	// Append the values (country codes) to the args slice
	for _, value := range values {
		*args = append(*args, value)
	}

	// Return the updated argument index
	return startIndex + len(values), nil
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

// BuildInCondition parses a comma-separated string of integers and appends a SQL IN condition
// to the provided conditions and args slices. It returns the updated SQL argument index.
//
// Parameters:
//   - intStr: A comma-separated list of integers (e.g., "1,2,3").
//   - format: A SQL format string containing a single %s for the placeholder list.
//   - label: A label used for error messages.
//   - startIndex: The current SQL parameter index (e.g., 1 for $1).
//   - conditions: A pointer to the slice of condition strings to append to.
//   - args: A pointer to the slice of SQL arguments to append to.
//
// Returns:
//   - newIndex: The next available argument index after this condition.
//   - err: An error if input parsing fails.
func BuildInCondition(intStr, format, label string, startIndex int, conditions *[]string, args *[]interface{}) (int, error) {
	if intStr == "" {
		return startIndex, nil
	}

	parts := strings.Split(intStr, ",")
	var integers []int
	for _, part := range parts {
		num, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return startIndex, fmt.Errorf("%s format error: %s", label, intStr)
		}
		integers = append(integers, num)
	}

	var placeholders []string
	for i := range integers {
		placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
	}

	condition := fmt.Sprintf(format, strings.Join(placeholders, ", "))
	*conditions = append(*conditions, condition)

	for _, v := range integers {
		*args = append(*args, v)
	}

	return startIndex + len(integers), nil
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
// that the pattern is safe for use in SQL queries.
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

// BuildGeographicRadiusCondition adds a PostGIS ST_DWithin condition to the given conditions slice,
// if all three input strings (lonStr, latStr, radiusStr) are valid and non-empty.
//
// Parameters:
//   - lonStr, latStr, radiusStr: string inputs for longitude, latitude, and radius in meters.
//   - columnExpr: the SQL expression for the geometry column (e.g., "v.wkb_geometry").
//   - startIndex: the starting placeholder index for the SQL arguments.
//   - conditions: a pointer to the slice of WHERE conditions to append to.
//   - args: a pointer to the slice of SQL arguments.
//
// Returns:
//   - nextArgIndex: the next available argument index after this condition.
//   - err: any error encountered while parsing the inputs.
func BuildGeographicRadiusCondition(
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
		return startIndex, fmt.Errorf("longitude '%s' is invalid", lonStr)
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return startIndex, fmt.Errorf("latitude '%s' is invalid", latStr)
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
