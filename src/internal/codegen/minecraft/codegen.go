package minecraft

import (
	"bytes"
	"encoding/json"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/patterns"
	"github.com/puff-lang/puff/internal/project"
)

type File struct {
	Path string
	Data []byte
}

type Output struct {
	Files []File
}

type Result struct {
	Output      Output
	Diagnostics []diagnostic.Diagnostic
}

type resourceLocation struct {
	namespace string
	path      string
}

func (resource resourceLocation) String() string {
	return resource.namespace + ":" + resource.path
}

type moduleInfo struct {
	namespace string
	path      string
	source    ir.SourceRef
}

type generator struct {
	program     *ir.Project
	config      project.Config
	target      targetSpec
	modules     map[string]moduleInfo
	functions   map[ir.SymbolID]ir.Function
	resources   map[ir.SymbolID]resourceLocation
	resourceIDs map[string]ir.SymbolID
	files       map[string][]byte
	diagnostics []diagnostic.Diagnostic
}

func Generate(program *ir.Project, config project.Config) Result {
	target, issue := resolveTarget(config.Minecraft)
	if issue != nil {
		return Result{Diagnostics: []diagnostic.Diagnostic{*issue}}
	}
	if program == nil {
		return Result{Diagnostics: []diagnostic.Diagnostic{codegenDiagnostic(ir.SourceRef{}, "project is nil")}}
	}
	if !validNamespace(config.Pack.ID) {
		return Result{Diagnostics: []diagnostic.Diagnostic{namespaceDiagnostic(ir.SourceRef{})}}
	}

	current := &generator{
		program:     program,
		config:      config,
		target:      target,
		modules:     make(map[string]moduleInfo),
		functions:   make(map[ir.SymbolID]ir.Function),
		resources:   make(map[ir.SymbolID]resourceLocation),
		resourceIDs: make(map[string]ir.SymbolID),
		files:       make(map[string][]byte),
	}
	current.indexModules()
	current.indexFunctions()
	if len(current.diagnostics) != 0 {
		return Result{Diagnostics: current.diagnostics}
	}

	description := config.Pack.Name
	if description == "" {
		description = config.Pack.ID
	}
	metadata, err := encodeJSON(struct {
		Pack struct {
			PackFormat  int    `json:"pack_format"`
			Description string `json:"description"`
		} `json:"pack"`
	}{Pack: struct {
		PackFormat  int    `json:"pack_format"`
		Description string `json:"description"`
	}{PackFormat: target.PackFormat, Description: description}})
	if err != nil {
		return Result{Diagnostics: []diagnostic.Diagnostic{codegenDiagnostic(ir.SourceRef{}, err.Error())}}
	}
	current.addFile("pack.mcmeta", metadata, ir.SourceRef{})
	current.generateFunctions()
	current.generateTagsAndBootstrap()
	if len(current.diagnostics) != 0 {
		return Result{Diagnostics: current.diagnostics}
	}

	paths := make([]string, 0, len(current.files))
	for filePath := range current.files {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	output := Output{Files: make([]File, 0, len(paths))}
	for _, filePath := range paths {
		output.Files = append(output.Files, File{Path: filePath, Data: append([]byte(nil), current.files[filePath]...)})
	}
	return Result{Output: output}
}

func (current *generator) indexModules() {
	for _, module := range current.program.Modules {
		modulePath, ok := moduleResourcePath(module.Path)
		if !ok {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(module.Source))
			continue
		}
		namespace := module.Namespace
		if namespace == "" {
			namespace = current.config.Pack.ID
		}
		if !validNamespace(namespace) {
			current.diagnostics = append(current.diagnostics, namespaceDiagnostic(module.Source))
			continue
		}
		if _, exists := current.modules[module.Path]; exists {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(module.Source))
			continue
		}
		current.modules[module.Path] = moduleInfo{namespace: namespace, path: modulePath, source: module.Source}
	}
}

func (current *generator) indexFunctions() {
	for _, function := range current.program.Functions {
		module, ok := current.modules[function.ID.Module]
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(function.Source, "function references an unknown module"))
			continue
		}
		name, ok := normalizeResourcePath(function.ID.Name)
		if !ok {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(function.Source))
			continue
		}
		resource := resourceLocation{namespace: module.namespace, path: module.path + "/" + name}
		key := resource.String()
		if previous, exists := current.resourceIDs[key]; exists && previous != function.ID {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(function.Source))
			continue
		}
		if _, exists := current.functions[function.ID]; exists {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(function.Source))
			continue
		}
		current.functions[function.ID] = function
		current.resources[function.ID] = resource
		current.resourceIDs[key] = function.ID
	}
}

func (current *generator) generateFunctions() {
	ids := make([]ir.SymbolID, 0, len(current.functions))
	for id := range current.functions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(left, right int) bool {
		return current.resources[ids[left]].String() < current.resources[ids[right]].String()
	})
	for _, id := range ids {
		function := current.functions[id]
		data, ok := current.renderFunction(function)
		if ok {
			resource := current.resources[id]
			current.addFile("data/"+resource.namespace+"/function/"+resource.path+".mcfunction", data, function.Source)
		}
	}
}

func (current *generator) renderFunction(function ir.Function) ([]byte, bool) {
	if len(function.Parameters) != 0 {
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(function.Source, "function parameters are not supported"))
		return nil, false
	}
	lines := make([]string, 0, len(function.Commands))
	for _, command := range function.Commands {
		switch node := command.(type) {
		case *ir.Return:
			line, ok := current.renderReturn(function, node)
			if !ok {
				return nil, false
			}
			lines = append(lines, line)
		case *ir.Effect:
			generated, ok := current.renderEffect(node)
			if !ok {
				return nil, false
			}
			lines = append(lines, generated...)
		default:
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(function.Source, "unsupported command"))
			return nil, false
		}
	}
	if len(lines) == 0 {
		return []byte{}, true
	}
	return []byte(strings.Join(lines, "\n") + "\n"), true
}

func (current *generator) renderReturn(function ir.Function, command *ir.Return) (string, bool) {
	if command == nil {
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(function.Source, "nil return command"))
		return "", false
	}
	if _, ok := command.Value.(*ir.Nil); ok {
		return "return 0", true
	}
	value, ok := current.staticSNBT(command.Value)
	if !ok {
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(command.Source, "return value is not supported for Minecraft "+current.target.Version))
		return "", false
	}
	resource := current.resources[function.ID]
	storagePath := storagePath("returns", resource)
	return "return run data modify storage " + current.config.Pack.ID + ":puff_runtime " + storagePath + " set value " + value, true
}

func (current *generator) renderEffect(effect *ir.Effect) ([]string, bool) {
	if effect == nil || effect.PatternID != patterns.CoreSendEffectID {
		ref := ir.SourceRef{}
		if effect != nil {
			ref = effect.Source
		}
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(ref, "unsupported effect"))
		return nil, false
	}
	arguments := make(map[string]ir.Value, len(effect.Arguments))
	for _, argument := range effect.Arguments {
		if _, exists := arguments[argument.Name]; exists {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(effect.Source, "duplicate effect argument"))
			return nil, false
		}
		arguments[argument.Name] = argument.Value
	}
	text, textOK := arguments["text"].(*ir.Text)
	target, targetOK := arguments["target"].(*ir.Reference)
	if !textOK || !targetOK || len(arguments) != 2 {
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(effect.Source, "invalid send arguments"))
		return nil, false
	}

	switch target.Name {
	case "player":
		components, prelude, ok := current.renderText(text)
		if !ok {
			return nil, false
		}
		data, err := encodeCompactJSON(components)
		if err != nil {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(effect.Source, err.Error()))
			return nil, false
		}
		return append(prelude, "tellraw @a "+string(data)), true
	case "console":
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(effect.Source, "console output is not supported"))
		return nil, false
	default:
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(target.Source, "unsupported send target"))
		return nil, false
	}
}

func (current *generator) renderText(text *ir.Text) ([]json.RawMessage, []string, bool) {
	components := make([]json.RawMessage, 0, len(text.Parts))
	prelude := make([]string, 0)
	for _, part := range text.Parts {
		switch node := part.(type) {
		case *ir.TextLiteral:
			component, _ := encodeCompactJSON(struct {
				Text string `json:"text"`
			}{Text: node.Value})
			components = append(components, component)
		case *ir.TextInterpolation:
			component, commands, ok := current.renderInterpolation(node)
			if !ok {
				return nil, nil, false
			}
			components = append(components, component)
			prelude = append(prelude, commands...)
		default:
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(text.Source, "unsupported text part"))
			return nil, nil, false
		}
	}
	return components, prelude, true
}

func (current *generator) renderInterpolation(interpolation *ir.TextInterpolation) (json.RawMessage, []string, bool) {
	if interpolation == nil {
		current.diagnostics = append(current.diagnostics, codegenDiagnostic(ir.SourceRef{}, "nil interpolation"))
		return nil, nil, false
	}
	switch value := interpolation.Value.(type) {
	case *ir.Call:
		if len(value.Arguments) != 0 {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(value.Source, "call arguments are not supported"))
			return nil, nil, false
		}
		resource, ok := current.resources[value.Function]
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(value.Source, "call references an unknown function"))
			return nil, nil, false
		}
		component, _ := encodeCompactJSON(struct {
			NBT       string `json:"nbt"`
			Storage   string `json:"storage"`
			Interpret bool   `json:"interpret"`
		}{
			NBT:       storagePath("returns", resource),
			Storage:   current.config.Pack.ID + ":puff_runtime",
			Interpret: false,
		})
		return component, []string{"function " + resource.String()}, true
	case *ir.Reference:
		if value.Symbol.Module == "" || value.Symbol.Name == "" {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(value.Source, "local references are not supported"))
			return nil, nil, false
		}
		storagePath, ok := current.globalStoragePath(value.Symbol)
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(value.Source, "reference points to an unknown global"))
			return nil, nil, false
		}
		component, _ := encodeCompactJSON(struct {
			NBT       string `json:"nbt"`
			Storage   string `json:"storage"`
			Interpret bool   `json:"interpret"`
		}{NBT: storagePath, Storage: current.config.Pack.ID + ":puff_runtime", Interpret: false})
		return component, nil, true
	default:
		plain, ok := staticText(value)
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(interpolation.Source, "unsupported interpolation"))
			return nil, nil, false
		}
		component, _ := encodeCompactJSON(struct {
			Text string `json:"text"`
		}{Text: plain})
		return component, nil, true
	}
}

func (current *generator) generateTagsAndBootstrap() {
	tags := map[string][]string{"load": {}, "tick": {}}
	for _, tag := range current.program.Tags {
		if tag.Name != "load" && tag.Name != "tick" {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(ir.SourceRef{}, "unsupported function tag"))
			continue
		}
		for _, id := range tag.Functions {
			resource, ok := current.resources[id]
			if !ok {
				current.diagnostics = append(current.diagnostics, codegenDiagnostic(ir.SourceRef{}, "tag references an unknown function"))
				continue
			}
			tags[tag.Name] = append(tags[tag.Name], resource.String())
		}
	}
	if len(current.diagnostics) != 0 {
		return
	}

	loadFunctions := sortedUnique(tags["load"])
	if len(current.program.Globals) != 0 || len(loadFunctions) != 0 {
		bootstrap := resourceLocation{namespace: current.config.Pack.ID, path: "__puff/load"}
		lines, ok := current.renderGlobalInitializers()
		if !ok {
			return
		}
		for _, function := range loadFunctions {
			lines = append(lines, "function "+function)
		}
		data := []byte{}
		if len(lines) != 0 {
			data = []byte(strings.Join(lines, "\n") + "\n")
		}
		current.addFile("data/"+bootstrap.namespace+"/function/"+bootstrap.path+".mcfunction", data, ir.SourceRef{})
		tags["load"] = []string{bootstrap.String()}
	}

	for _, name := range []string{"load", "tick"} {
		values := sortedUnique(tags[name])
		if len(values) == 0 {
			continue
		}
		data, err := encodeJSON(struct {
			Values []string `json:"values"`
		}{Values: values})
		if err != nil {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(ir.SourceRef{}, err.Error()))
			return
		}
		current.addFile("data/minecraft/tags/function/"+name+".json", data, ir.SourceRef{})
	}
}

func (current *generator) renderGlobalInitializers() ([]string, bool) {
	type initializer struct {
		path   string
		value  string
		source ir.SourceRef
	}
	initializers := make([]initializer, 0, len(current.program.Globals))
	seen := make(map[string]struct{})
	for _, global := range current.program.Globals {
		storagePath, ok := current.globalStoragePath(global.ID)
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(global.Source, "global references an unknown module"))
			continue
		}
		if _, exists := seen[storagePath]; exists {
			current.diagnostics = append(current.diagnostics, resourceDiagnostic(global.Source))
			continue
		}
		seen[storagePath] = struct{}{}
		value, ok := current.staticSNBT(global.Initializer)
		if !ok {
			current.diagnostics = append(current.diagnostics, codegenDiagnostic(global.Source, "global initializer is not supported for Minecraft "+current.target.Version))
			continue
		}
		initializers = append(initializers, initializer{path: storagePath, value: value, source: global.Source})
	}
	if len(current.diagnostics) != 0 {
		return nil, false
	}
	sort.Slice(initializers, func(left, right int) bool { return initializers[left].path < initializers[right].path })
	lines := make([]string, 0, len(initializers))
	for _, initializer := range initializers {
		storage := current.config.Pack.ID + ":puff_runtime"
		lines = append(lines, "execute unless data storage "+storage+" "+initializer.path+" run data modify storage "+storage+" "+initializer.path+" set value "+initializer.value)
	}
	return lines, true
}

func (current *generator) globalStoragePath(id ir.SymbolID) (string, bool) {
	module, ok := current.modules[id.Module]
	if !ok {
		return "", false
	}
	name, ok := normalizeResourcePath(id.Name)
	if !ok || strings.Contains(name, "/") {
		return "", false
	}
	return storagePath("globals", resourceLocation{namespace: module.namespace, path: module.path + "/" + name}), true
}

func (current *generator) addFile(filePath string, data []byte, source ir.SourceRef) {
	if _, err := portablePathKey(filePath); err != nil {
		current.diagnostics = append(current.diagnostics, resourceDiagnostic(source))
		return
	}
	if _, exists := current.files[filePath]; exists {
		current.diagnostics = append(current.diagnostics, resourceDiagnostic(source))
		return
	}
	current.files[filePath] = append([]byte(nil), data...)
}

func moduleResourcePath(modulePath string) (string, bool) {
	if modulePath == "" || strings.ContainsAny(modulePath, "\\:") || strings.HasPrefix(modulePath, "/") || path.Clean(modulePath) != modulePath || path.Ext(modulePath) != ".puff" {
		return "", false
	}
	return normalizeResourcePath(strings.TrimSuffix(modulePath, ".puff"))
}

func (current *generator) staticSNBT(value ir.Value) (string, bool) {
	switch node := value.(type) {
	case *ir.Bool:
		if node.Value {
			return "1b", true
		}
		return "0b", true
	case *ir.Int:
		return strconv.FormatInt(node.Value, 10), true
	case *ir.Float:
		return strconv.FormatFloat(node.Value, 'g', -1, 64) + "d", true
	case *ir.Text:
		text, ok := plainText(node)
		if !ok {
			return "", false
		}
		return quoteSNBTString(text, current.target.Version)
	default:
		return "", false
	}
}

func storagePath(root string, resource resourceLocation) string {
	return root + ".\"" + resource.String() + "\""
}

func quoteSNBTString(value, target string) (string, bool) {
	escapedControls := compareVersions(mustParseVersion(target), []int{1, 21, 5}) >= 0
	var output strings.Builder
	output.WriteByte('"')
	for _, char := range value {
		switch char {
		case '"', '\\':
			output.WriteByte('\\')
			output.WriteRune(char)
		case '\n':
			if !escapedControls {
				return "", false
			}
			output.WriteString(`\n`)
		case '\r':
			if !escapedControls {
				return "", false
			}
			output.WriteString(`\r`)
		case '\t':
			if !escapedControls {
				return "", false
			}
			output.WriteString(`\t`)
		default:
			if char < 0x20 || char == 0x7f {
				return "", false
			}
			output.WriteRune(char)
		}
	}
	output.WriteByte('"')
	return output.String(), true
}

func mustParseVersion(value string) []int {
	version, _ := parseVersion(value)
	return version
}

func staticText(value ir.Value) (string, bool) {
	switch node := value.(type) {
	case *ir.Nil:
		return "nil", true
	case *ir.Bool:
		return strconv.FormatBool(node.Value), true
	case *ir.Int:
		return strconv.FormatInt(node.Value, 10), true
	case *ir.Float:
		return strconv.FormatFloat(node.Value, 'g', -1, 64), true
	case *ir.Text:
		return plainText(node)
	default:
		return "", false
	}
}

func plainText(text *ir.Text) (string, bool) {
	if text == nil {
		return "", false
	}
	var output strings.Builder
	for _, part := range text.Parts {
		literal, ok := part.(*ir.TextLiteral)
		if !ok {
			return "", false
		}
		output.WriteString(literal.Value)
	}
	return output.String(), true
}

func encodeJSON(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func encodeCompactJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	write := 0
	for _, value := range result {
		if write == 0 || result[write-1] != value {
			result[write] = value
			write++
		}
	}
	return result[:write]
}

func namespaceDiagnostic(source ir.SourceRef) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidNamespace,
		Phase:    diagnostic.PhaseCodegen,
		Severity: diagnostic.SeverityError,
		Message:  "Invalid namespace.",
		File:     source.File,
		Span:     source.Span,
	}
}

func resourceDiagnostic(source ir.SourceRef) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidMinecraftResource,
		Phase:    diagnostic.PhaseCodegen,
		Severity: diagnostic.SeverityError,
		Message:  "Invalid Minecraft resource location.",
		File:     source.File,
		Span:     source.Span,
	}
}

func codegenDiagnostic(source ir.SourceRef, detail string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeCodegenError,
		Phase:    diagnostic.PhaseCodegen,
		Severity: diagnostic.SeverityError,
		Message:  "Failed to generate datapack.",
		Hint:     detail,
		File:     source.File,
		Span:     source.Span,
	}
}
