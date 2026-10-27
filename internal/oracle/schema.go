package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/c360studio/semvocab/internal/ledger"
)

// SchemaBundle names a vendored JSON Schema bundle by its key under the
// ledger's pins.schemas.
type SchemaBundle string

// The vendored OGC Draft 2020-12 JSON Schema bundles.
const (
	SWE30      SchemaBundle = "swe-3.0"
	SensorML30 SchemaBundle = "sensorml-3.0"
)

var schemaDirs = map[SchemaBundle]string{
	SWE30:      "ogcapi-connected-systems",
	SensorML30: "ogcapi-connected-systems",
}

var domainSchemaPrefixes = map[SchemaBundle]string{
	SWE30:      "swecommon/schemas/json/",
	SensorML30: "sensorml/schemas/json/",
}

// SchemaRoot is the absolute path of a schema bundle's vendored tree.
func SchemaRoot(b SchemaBundle) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", schemaDirs[b])
}

// ReadSchemaFile returns the bytes of one vendored schema bundle file, named by
// its path in the source repository, after checking its sha256 against
// pins.schemas. A path the ledger does not list fails before any read.
func ReadSchemaFile(b SchemaBundle, p string) ([]byte, error) {
	l, err := ledger.Load()
	if err != nil {
		return nil, err
	}
	pin, ok := l.Pins.Schemas[string(b)]
	if !ok {
		return nil, fmt.Errorf("oracle: ledger pins no schema bundle %q", b)
	}
	return readSchemaFile(SchemaRoot(b), pin, p)
}

func readSchemaFile(root string, pin ledger.SchemaBundle, p string) ([]byte, error) {
	f, ok := pin.File(p)
	if !ok || !filepath.IsLocal(filepath.FromSlash(p)) || path.Clean(p) != p {
		return nil, fmt.Errorf("oracle: schema %s is not listed in pins.schemas", p)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return nil, fmt.Errorf("oracle: %w", err)
	}
	if err := VerifySHA256(data, f.SHA256); err != nil {
		return nil, fmt.Errorf("oracle: schema %s: %w", p, err)
	}
	return data, nil
}

type schemaNode struct {
	Type       string                 `json:"type"`
	Ref        string                 `json:"$ref"`
	Const      string                 `json:"const"`
	Properties map[string]*schemaNode `json:"properties"`
	Defs       map[string]*schemaNode `json:"$defs"`
	AllOf      []*schemaNode          `json:"allOf"`
	OneOf      []*schemaNode          `json:"oneOf"`
	AnyOf      []*schemaNode          `json:"anyOf"`
	Items      *schemaNode            `json:"items"`
}

// SchemaCatalog is a parsed JSON Schema bundle indexed by upstream file path,
// used as an independent oracle for catalog/swe and catalog/sensorml tests.
type SchemaCatalog struct {
	bundle      SchemaBundle
	domainDir   string
	sweNS       string
	smlNS       string
	docs        map[string]*schemaNode
	parentByIRI map[string]string
}

var schemaCache sync.Map // SchemaBundle -> *schemaLoadResult

type schemaLoadResult struct {
	once sync.Once
	cat  *SchemaCatalog
	err  error
}

// LoadSchemaCatalog parses every JSON Schema file in bundle b after verifying
// each sha256 against pins.schemas. For SensorML30, SWE30 is also loaded so
// cross-bundle $ref links into swecommon/schemas/json/ resolve cleanly.
func LoadSchemaCatalog(b SchemaBundle) (*SchemaCatalog, error) {
	v, _ := schemaCache.LoadOrStore(b, &schemaLoadResult{})
	res := v.(*schemaLoadResult)
	res.once.Do(func() { res.cat, res.err = loadSchemaCatalog(b) })
	return res.cat, res.err
}

func loadSchemaCatalog(b SchemaBundle) (*SchemaCatalog, error) {
	domainDir, ok := domainSchemaPrefixes[b]
	if !ok {
		return nil, fmt.Errorf("oracle: unknown schema bundle %q", b)
	}
	l, err := ledger.Load()
	if err != nil {
		return nil, err
	}
	pin, ok := l.Pins.Schemas[string(b)]
	if !ok {
		return nil, fmt.Errorf("oracle: ledger pins no schema bundle %q", b)
	}
	sweEntry, err := l.Entry("swe")
	if err != nil {
		return nil, err
	}
	smlEntry, err := l.Entry("sensorml")
	if err != nil {
		return nil, err
	}
	docs := map[string]*schemaNode{}
	if b == SensorML30 {
		sweCat, err := LoadSchemaCatalog(SWE30)
		if err != nil {
			return nil, err
		}
		for k, v := range sweCat.docs {
			docs[k] = v
		}
	}
	for _, f := range pin.Files {
		if !strings.HasSuffix(f.Path, ".json") {
			continue
		}
		data, err := ReadSchemaFile(b, f.Path)
		if err != nil {
			return nil, err
		}
		var root schemaNode
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("oracle: parse %s: %w", f.Path, err)
		}
		docs[f.Path] = &root
	}
	cat := &SchemaCatalog{
		bundle:      b,
		domainDir:   domainDir,
		sweNS:       sweEntry.Namespace.IRI,
		smlNS:       smlEntry.Namespace.IRI,
		docs:        docs,
		parentByIRI: map[string]string{},
	}
	if err := cat.indexParents(); err != nil {
		return nil, err
	}
	return cat, nil
}

func (c *SchemaCatalog) indexParents() error {
	for fp, doc := range c.docs {
		ns := c.namespaceForFile(fp)
		if ns == "" {
			continue
		}
		local := strings.TrimSuffix(path.Base(fp), ".json")
		if !isPureRefAlias(doc) && c.resolvesToObject(doc, fp) {
			p, err := c.extractParentIRI(doc, fp)
			if err != nil {
				return err
			}
			if p != "" {
				c.parentByIRI[ns+local] = p
			}
		}
		for defName, defNode := range doc.Defs {
			if isPureRefAlias(defNode) || !c.resolvesToObject(defNode, fp) {
				continue
			}
			p, err := c.extractParentIRI(defNode, fp)
			if err != nil {
				return err
			}
			if p != "" {
				c.parentByIRI[ns+defName] = p
			}
		}
	}
	return nil
}

func (c *SchemaCatalog) namespaceForFile(fp string) string {
	switch {
	case strings.HasPrefix(fp, domainSchemaPrefixes[SWE30]):
		return c.sweNS
	case strings.HasPrefix(fp, domainSchemaPrefixes[SensorML30]):
		return c.smlNS
	default:
		return ""
	}
}

func isPureRefAlias(n *schemaNode) bool {
	return n.Ref != "" && n.Type == "" && len(n.Properties) == 0 &&
		len(n.AllOf) == 0 && len(n.OneOf) == 0 && len(n.AnyOf) == 0
}

func resolveRefLocation(baseFile, ref string) (filePath, defName string, ok bool) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return "", "", false
	}
	filePart, frag, hasFrag := strings.Cut(ref, "#")
	targetFile := baseFile
	if filePart != "" {
		targetFile = path.Clean(path.Join(path.Dir(baseFile), filePart))
	}
	if !hasFrag {
		return targetFile, "", true
	}
	if !strings.HasPrefix(frag, "/$defs/") {
		return "", "", false
	}
	return targetFile, strings.TrimPrefix(frag, "/$defs/"), true
}

func (c *SchemaCatalog) resolveRef(baseFile, ref string) (*schemaNode, string, string, bool) {
	curFile, curDef, ok := resolveRefLocation(baseFile, ref)
	if !ok {
		return nil, "", "", false
	}
	seen := map[string]bool{}
	for {
		key := curFile + "#" + curDef
		if seen[key] {
			return nil, "", "", false
		}
		seen[key] = true
		doc := c.docs[curFile]
		if doc == nil {
			return nil, "", "", false
		}
		node := doc
		local := strings.TrimSuffix(path.Base(curFile), ".json")
		if curDef != "" {
			node = doc.Defs[curDef]
			local = curDef
			if node == nil {
				return nil, "", "", false
			}
		}
		if !isPureRefAlias(node) {
			return node, curFile, local, true
		}
		curFile, curDef, ok = resolveRefLocation(curFile, node.Ref)
		if !ok {
			return nil, "", "", false
		}
	}
}

func (c *SchemaCatalog) resolvesToObject(n *schemaNode, filePath string) bool {
	if n == nil {
		return false
	}
	if n.Type == "object" {
		return true
	}
	if n.Type != "" {
		return false
	}
	if len(n.Properties) > 0 && n.Ref == "" {
		return true
	}
	if n.Ref != "" {
		target, targetFile, _, ok := c.resolveRef(filePath, n.Ref)
		return ok && c.resolvesToObject(target, targetFile)
	}
	for _, sub := range n.AllOf {
		if c.resolvesToObject(sub, filePath) {
			return true
		}
	}
	return false
}

func findConstTypeBranch(n *schemaNode, localName string) *schemaNode {
	if n == nil {
		return nil
	}
	if tp := n.Properties["type"]; tp != nil && tp.Const == localName {
		return n
	}
	for _, sub := range n.OneOf {
		if found := findConstTypeBranch(sub, localName); found != nil {
			return found
		}
	}
	for _, sub := range n.AllOf {
		if found := findConstTypeBranch(sub, localName); found != nil {
			return found
		}
	}
	for _, sub := range n.AnyOf {
		if found := findConstTypeBranch(sub, localName); found != nil {
			return found
		}
	}
	if found := findConstTypeBranch(n.Items, localName); found != nil {
		return found
	}
	for _, def := range n.Defs {
		if found := findConstTypeBranch(def, localName); found != nil {
			return found
		}
	}
	return nil
}

func (c *SchemaCatalog) locateObjectNode(localName, location string) (*schemaNode, string, bool) {
	filePath, frag, hasFrag := strings.Cut(location, "#")
	if !strings.HasPrefix(filePath, c.domainDir) {
		return nil, "", false
	}
	doc := c.docs[filePath]
	if doc == nil {
		return nil, "", false
	}
	if hasFrag {
		if !strings.HasPrefix(frag, "/$defs/") {
			return nil, "", false
		}
		defName := strings.TrimPrefix(frag, "/$defs/")
		if defName != localName {
			return nil, "", false
		}
		node := doc.Defs[defName]
		if node == nil || isPureRefAlias(node) || !c.resolvesToObject(node, filePath) {
			return nil, "", false
		}
		return node, filePath, true
	}
	if strings.TrimSuffix(path.Base(filePath), ".json") == localName {
		if isPureRefAlias(doc) || !c.resolvesToObject(doc, filePath) {
			return nil, "", false
		}
		return doc, filePath, true
	}
	branch := findConstTypeBranch(doc, localName)
	if branch == nil || !c.resolvesToObject(branch, filePath) {
		return nil, "", false
	}
	return branch, filePath, true
}

func (c *SchemaCatalog) extractParentIRI(node *schemaNode, filePath string) (string, error) {
	var parentIRI string
	for _, branch := range node.AllOf {
		if branch.Ref == "" {
			continue
		}
		_, targetFile, targetLocal, ok := c.resolveRef(filePath, branch.Ref)
		if !ok {
			return "", fmt.Errorf("oracle: unresolved allOf $ref %q in %s", branch.Ref, filePath)
		}
		if targetLocal == "SoftNamedProperty" {
			continue
		}
		ns := c.namespaceForFile(targetFile)
		if ns == "" {
			return "", fmt.Errorf("oracle: allOf parent %s in %s lies outside swe and sensorml", targetFile, filePath)
		}
		cand := ns + targetLocal
		if parentIRI != "" && parentIRI != cand {
			return "", fmt.Errorf("oracle: multiple allOf object parents (%s, %s) in %s", parentIRI, cand, filePath)
		}
		parentIRI = cand
	}
	return parentIRI, nil
}

// HasObjectClass reports whether location (<file> or <file>#/$defs/<def>) in
// bundle c resolves to an object schema for localName.
func (c *SchemaCatalog) HasObjectClass(localName, location string) bool {
	_, _, ok := c.locateObjectNode(localName, location)
	return ok
}

// DirectParent returns the asserted allOf object $ref parent of localName at
// location (as a local name within the same bundle, or "swe:<LocalName>" when
// SensorML30 crosses into SWE30), or "" when no allOf object parent is present.
func (c *SchemaCatalog) DirectParent(localName, location string) (string, error) {
	node, filePath, ok := c.locateObjectNode(localName, location)
	if !ok {
		return "", fmt.Errorf("oracle: %s at %s is not an object class in %s", localName, location, c.bundle)
	}
	parentIRI, err := c.extractParentIRI(node, filePath)
	if err != nil || parentIRI == "" {
		return parentIRI, err
	}
	ownNS := c.sweNS
	if c.bundle == SensorML30 {
		ownNS = c.smlNS
	}
	if strings.HasPrefix(parentIRI, ownNS) {
		return strings.TrimPrefix(parentIRI, ownNS), nil
	}
	if c.bundle == SensorML30 && strings.HasPrefix(parentIRI, c.sweNS) {
		return "swe:" + strings.TrimPrefix(parentIRI, c.sweNS), nil
	}
	return "", fmt.Errorf("oracle: unexpected parent IRI %s for %s", parentIRI, localName)
}

// Closure returns the breadth-first allOf parent IRI closure of classIRI across
// the loaded schema bundle(s), directly from the parsed JSON Schema AST.
func (c *SchemaCatalog) Closure(classIRI string) []string {
	var out []string
	seen := map[string]bool{classIRI: true}
	for cur := classIRI; ; {
		p, ok := c.parentByIRI[cur]
		if !ok || seen[p] {
			return out
		}
		seen[p] = true
		out = append(out, p)
		cur = p
	}
}

// DeclaredProperties returns the sorted unique property keys inside
// "properties" maps across the bundle's own domain schema directory.
func (c *SchemaCatalog) DeclaredProperties() []string {
	seen := map[string]bool{}
	var walk func(*schemaNode)
	walk = func(n *schemaNode) {
		if n == nil {
			return
		}
		for prop, sub := range n.Properties {
			seen[prop] = true
			walk(sub)
		}
		for _, sub := range n.Defs {
			walk(sub)
		}
		for _, sub := range n.AllOf {
			walk(sub)
		}
		for _, sub := range n.OneOf {
			walk(sub)
		}
		for _, sub := range n.AnyOf {
			walk(sub)
		}
		walk(n.Items)
	}
	for fp, doc := range c.docs {
		if strings.HasPrefix(fp, c.domainDir) {
			walk(doc)
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
