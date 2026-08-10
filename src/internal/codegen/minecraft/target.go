package minecraft

import (
	"strconv"
	"strings"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/project"
)

type targetSpec struct {
	Version    string
	PackFormat int
}

var supportedTargets = []targetSpec{
	{Version: "1.21", PackFormat: 48},
	{Version: "1.21.1", PackFormat: 48},
	{Version: "1.21.2", PackFormat: 57},
	{Version: "1.21.3", PackFormat: 57},
	{Version: "1.21.4", PackFormat: 61},
	{Version: "1.21.5", PackFormat: 71},
	{Version: "1.21.6", PackFormat: 80},
}

func resolveTarget(config project.MinecraftConfig) (targetSpec, *diagnostic.Diagnostic) {
	if config.PackFormat < 0 {
		return targetSpec{}, invalidVersionDiagnostic()
	}

	constraints, ok := parseVersionConstraints(config.Versions)
	if !ok {
		return targetSpec{}, invalidVersionDiagnostic()
	}

	if config.Target != "" {
		if _, ok := parseVersion(config.Target); !ok {
			return targetSpec{}, invalidVersionDiagnostic()
		}

		target, found := findSupportedTarget(config.Target)
		if !found || !matchesConstraints(config.Target, constraints) {
			return targetSpec{}, unsupportedVersionDiagnostic()
		}
		if config.PackFormat > 0 && config.PackFormat != target.PackFormat {
			return targetSpec{}, invalidVersionDiagnostic()
		}
		return target, nil
	}

	for i := len(supportedTargets) - 1; i >= 0; i-- {
		target := supportedTargets[i]
		if !matchesConstraints(target.Version, constraints) {
			continue
		}
		if config.PackFormat > 0 && config.PackFormat != target.PackFormat {
			return targetSpec{}, invalidVersionDiagnostic()
		}
		return target, nil
	}

	return targetSpec{}, unsupportedVersionDiagnostic()
}

func validNamespace(namespace string) bool {
	if namespace == "" {
		return false
	}
	for _, char := range namespace {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func normalizeResourcePath(path string) (string, bool) {
	if path == "" || strings.HasPrefix(path, "/") || strings.ContainsAny(path, `\:`) {
		return "", false
	}

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}

		var normalized strings.Builder
		for index := 0; index < len(segment); index++ {
			char := segment[index]
			if !isResourceInputChar(char) {
				return "", false
			}
			if char >= 'A' && char <= 'Z' {
				previousLowerOrDigit := index > 0 && (isASCIILower(segment[index-1]) || isASCIIDigit(segment[index-1]))
				nextLower := index+1 < len(segment) && isASCIILower(segment[index+1])
				previousUpper := index > 0 && segment[index-1] >= 'A' && segment[index-1] <= 'Z'
				if normalized.Len() > 0 && (previousLowerOrDigit || previousUpper && nextLower) && segment[index-1] != '_' {
					normalized.WriteByte('_')
				}
				char += 'a' - 'A'
			}
			normalized.WriteByte(char)
		}

		segments[i] = normalized.String()
	}

	return strings.Join(segments, "/"), true
}

type versionConstraint struct {
	operator string
	version  []int
}

func parseVersionConstraints(value string) ([]versionConstraint, bool) {
	if strings.TrimSpace(value) == "" {
		return nil, true
	}

	tokens := strings.Fields(value)
	constraints := make([]versionConstraint, 0, len(tokens))
	for _, token := range tokens {
		operator := "="
		versionText := token
		hasOperator := false
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(token, candidate) {
				operator = candidate
				versionText = strings.TrimPrefix(token, candidate)
				hasOperator = true
				break
			}
		}
		if len(tokens) > 1 && !hasOperator {
			return nil, false
		}

		version, ok := parseVersion(versionText)
		if !ok {
			return nil, false
		}
		constraints = append(constraints, versionConstraint{operator: operator, version: version})
	}

	return constraints, true
}

func parseVersion(value string) ([]int, bool) {
	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, false
	}

	version := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return nil, false
			}
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		version[i] = number
	}
	return version, true
}

func findSupportedTarget(version string) (targetSpec, bool) {
	for _, target := range supportedTargets {
		if target.Version == version {
			return target, true
		}
	}
	return targetSpec{}, false
}

func matchesConstraints(version string, constraints []versionConstraint) bool {
	parsed, ok := parseVersion(version)
	if !ok {
		return false
	}
	for _, constraint := range constraints {
		comparison := compareVersions(parsed, constraint.version)
		if constraint.operator == "=" && comparison != 0 ||
			constraint.operator == ">" && comparison <= 0 ||
			constraint.operator == "<" && comparison >= 0 ||
			constraint.operator == ">=" && comparison < 0 ||
			constraint.operator == "<=" && comparison > 0 {
			return false
		}
	}
	return true
}

func compareVersions(left, right []int) int {
	for i := 0; i < 3; i++ {
		var leftPart, rightPart int
		if i < len(left) {
			leftPart = left[i]
		}
		if i < len(right) {
			rightPart = right[i]
		}
		if leftPart < rightPart {
			return -1
		}
		if leftPart > rightPart {
			return 1
		}
	}
	return 0
}

func isResourceInputChar(char byte) bool {
	return isASCIILower(char) || char >= 'A' && char <= 'Z' || isASCIIDigit(char) || char == '.' || char == '_' || char == '-'
}

func isASCIILower(char byte) bool {
	return char >= 'a' && char <= 'z'
}

func isASCIIDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func invalidVersionDiagnostic() *diagnostic.Diagnostic {
	return &diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidMinecraftVersion,
		Phase:    diagnostic.PhaseProject,
		Severity: diagnostic.SeverityError,
		Message:  "Invalid Minecraft version.",
	}
}

func unsupportedVersionDiagnostic() *diagnostic.Diagnostic {
	return &diagnostic.Diagnostic{
		Code:     diagnostic.CodeUnsupportedMinecraftVersion,
		Phase:    diagnostic.PhaseCodegen,
		Severity: diagnostic.SeverityError,
		Message:  "Unsupported Minecraft version.",
	}
}
