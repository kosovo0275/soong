package android

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/blueprint"
)

const (
	namespacePrefix = "//"
	modulePrefix    = ":"
)

func init() {
	RegisterModuleType("soong_namespace", NamespaceFactory)
}

type sortedNamespaces struct {
	lock   sync.Mutex
	items  []*Namespace
	sorted bool
}

func (s *sortedNamespaces) add(namespace *Namespace) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.sorted {
		panic("It is not supported to call sortedNamespaces.add() after sortedNamespaces.sortedItems()")
	}
	s.items = append(s.items, namespace)
}

func (s *sortedNamespaces) sortedItems() []*Namespace {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.sorted {
		less := func(i int, j int) bool {
			return s.items[i].Path < s.items[j].Path
		}
		sort.Slice(s.items, less)
		s.sorted = true
	}
	return s.items
}

func (s *sortedNamespaces) index(namespace *Namespace) int {
	for i, candidate := range s.sortedItems() {
		if namespace == candidate {
			return i
		}
	}
	return -1
}

type NameResolver struct {
	rootNamespace         *Namespace
	nextNamespaceId       int32
	sortedNamespaces      sortedNamespaces
	namespacesByDir       sync.Map
	namespaceExportFilter func(*Namespace) bool
}

func NewNameResolver(namespaceExportFilter func(*Namespace) bool) *NameResolver {
	namespacesByDir := sync.Map{}

	r := &NameResolver{
		namespacesByDir:       namespacesByDir,
		namespaceExportFilter: namespaceExportFilter,
	}
	r.rootNamespace = r.newNamespace(".")
	r.rootNamespace.visibleNamespaces = []*Namespace{r.rootNamespace}
	r.addNamespace(r.rootNamespace)

	return r
}

func (r *NameResolver) newNamespace(path string) *Namespace {
	namespace := NewNamespace(path)

	namespace.exportToKati = r.namespaceExportFilter(namespace)

	return namespace
}

func (r *NameResolver) addNewNamespaceForModule(module *NamespaceModule, path string) error {
	fileName := filepath.Base(path)
	if fileName != "Android.bp" {
		return errors.New("A namespace may only be declared in a file named Android.bp")
	}
	dir := filepath.Dir(path)

	namespace := r.newNamespace(dir)
	module.namespace = namespace
	module.resolver = r
	namespace.importedNamespaceNames = module.properties.Imports
	return r.addNamespace(namespace)
}

func (r *NameResolver) addNamespace(namespace *Namespace) (err error) {
	existingNamespace, exists := r.namespaceAt(namespace.Path)
	if exists {
		if existingNamespace.Path == namespace.Path {
			return fmt.Errorf("namespace %v already exists", namespace.Path)
		} else {
			return fmt.Errorf("a namespace must be the first module in the file")
		}
	}
	r.sortedNamespaces.add(namespace)

	r.namespacesByDir.Store(namespace.Path, namespace)
	return nil
}

func (r *NameResolver) namespaceAt(path string) (namespace *Namespace, found bool) {
	mapVal, found := r.namespacesByDir.Load(path)
	if !found {
		return nil, false
	}
	return mapVal.(*Namespace), true
}

func (r *NameResolver) findNamespace(path string) (namespace *Namespace) {
	namespace, found := r.namespaceAt(path)
	if found {
		return namespace
	}
	parentDir := filepath.Dir(path)
	if parentDir == path {
		return nil
	}
	namespace = r.findNamespace(parentDir)
	r.namespacesByDir.Store(path, namespace)
	return namespace
}

func (r *NameResolver) NewModule(ctx blueprint.NamespaceContext, moduleGroup blueprint.ModuleGroup, module blueprint.Module) (namespace blueprint.Namespace, errs []error) {
	newNamespace, ok := module.(*NamespaceModule)
	if ok {
		err := r.addNewNamespaceForModule(newNamespace, ctx.ModulePath())
		if err != nil {
			return nil, []error{err}
		}
		return nil, nil
	}

	ns := r.findNamespaceFromCtx(ctx)

	_, errs = ns.moduleContainer.NewModule(ctx, moduleGroup, module)
	if len(errs) > 0 {
		return nil, errs
	}

	amod, ok := module.(Module)
	if ok {
		amod.base().commonProperties.NamespaceExportedToMake = ns.exportToKati
	}

	return ns, nil
}

func (r *NameResolver) AllModules() []blueprint.ModuleGroup {
	childLists := [][]blueprint.ModuleGroup{}
	totalCount := 0
	for _, namespace := range r.sortedNamespaces.sortedItems() {
		newModules := namespace.moduleContainer.AllModules()
		totalCount += len(newModules)
		childLists = append(childLists, newModules)
	}

	allModules := make([]blueprint.ModuleGroup, 0, totalCount)
	for _, childList := range childLists {
		allModules = append(allModules, childList...)
	}
	return allModules
}

func (r *NameResolver) parseFullyQualifiedName(name string) (namespaceName string, moduleName string, ok bool) {
	if !strings.HasPrefix(name, namespacePrefix) {
		return "", "", false
	}
	name = strings.TrimPrefix(name, namespacePrefix)
	components := strings.Split(name, modulePrefix)
	if len(components) != 2 {
		return "", "", false
	}
	return components[0], components[1], true

}

func (r *NameResolver) getNamespacesToSearchForModule(sourceNamespace *Namespace) (searchOrder []*Namespace) {
	return sourceNamespace.visibleNamespaces
}

func (r *NameResolver) ModuleFromName(name string, namespace blueprint.Namespace) (group blueprint.ModuleGroup, found bool) {
	nsName, moduleName, isAbs := r.parseFullyQualifiedName(name)
	if isAbs {
		namespace, found := r.namespaceAt(nsName)
		if !found {
			return blueprint.ModuleGroup{}, false
		}
		container := namespace.moduleContainer
		return container.ModuleFromName(moduleName, nil)
	}
	for _, candidate := range r.getNamespacesToSearchForModule(namespace.(*Namespace)) {
		group, found = candidate.moduleContainer.ModuleFromName(name, nil)
		if found {
			return group, true
		}
	}
	return blueprint.ModuleGroup{}, false

}

func (r *NameResolver) Rename(oldName string, newName string, namespace blueprint.Namespace) []error {
	return namespace.(*Namespace).moduleContainer.Rename(oldName, newName, namespace)
}

func (r *NameResolver) FindNamespaceImports(namespace *Namespace) (err error) {
	namespace.visibleNamespaces = make([]*Namespace, 0, 2+len(namespace.importedNamespaceNames))
	namespace.visibleNamespaces = append(namespace.visibleNamespaces, namespace)
	for _, name := range namespace.importedNamespaceNames {
		imp, ok := r.namespaceAt(name)
		if !ok {
			return fmt.Errorf("namespace %v does not exist", name)
		}
		namespace.visibleNamespaces = append(namespace.visibleNamespaces, imp)
	}
	namespace.visibleNamespaces = append(namespace.visibleNamespaces, r.rootNamespace)
	return nil
}

func (r *NameResolver) chooseId(namespace *Namespace) {
	id := r.sortedNamespaces.index(namespace)
	if id < 0 {
		panic(fmt.Sprintf("Namespace not found: %v\n", namespace.id))
	}
	namespace.id = strconv.Itoa(id)
}

func (r *NameResolver) MissingDependencyError(depender string, dependerNamespace blueprint.Namespace, depName string) (err error) {
	text := fmt.Sprintf("%q depends on undefined module %q", depender, depName)

	_, _, isAbs := r.parseFullyQualifiedName(depName)
	if isAbs {
		return fmt.Errorf(text)
	}

	foundInNamespaces := []string{}
	for _, namespace := range r.sortedNamespaces.sortedItems() {
		_, found := namespace.moduleContainer.ModuleFromName(depName, nil)
		if found {
			foundInNamespaces = append(foundInNamespaces, namespace.Path)
		}
	}
	if len(foundInNamespaces) > 0 {
		dependerNs := dependerNamespace.(*Namespace)
		searched := r.getNamespacesToSearchForModule(dependerNs)
		importedNames := []string{}
		for _, ns := range searched {
			importedNames = append(importedNames, ns.Path)
		}
		text += fmt.Sprintf("\nModule %q is defined in namespace %q which can read these %v namespaces: %q", depender, dependerNs.Path, len(importedNames), importedNames)
		text += fmt.Sprintf("\nModule %q can be found in these namespaces: %q", depName, foundInNamespaces)
	}

	return fmt.Errorf(text)
}

func (r *NameResolver) GetNamespace(ctx blueprint.NamespaceContext) blueprint.Namespace {
	return r.findNamespaceFromCtx(ctx)
}

func (r *NameResolver) findNamespaceFromCtx(ctx blueprint.NamespaceContext) *Namespace {
	return r.findNamespace(filepath.Dir(ctx.ModulePath()))
}

func (r *NameResolver) UniqueName(ctx blueprint.NamespaceContext, name string) (unique string) {
	prefix := r.findNamespaceFromCtx(ctx).id
	if prefix != "" {
		prefix = prefix + "-"
	}
	return prefix + name
}

var _ blueprint.NameInterface = (*NameResolver)(nil)

type Namespace struct {
	blueprint.NamespaceMarker
	Path                   string
	importedNamespaceNames []string
	visibleNamespaces      []*Namespace
	id                     string
	exportToKati           bool
	moduleContainer        blueprint.NameInterface
}

func NewNamespace(path string) *Namespace {
	return &Namespace{Path: path, moduleContainer: blueprint.NewSimpleNameInterface()}
}

var _ blueprint.Namespace = (*Namespace)(nil)

type NamespaceModule struct {
	ModuleBase

	namespace *Namespace
	resolver  *NameResolver

	properties struct {
		Imports []string
	}
}

func (n *NamespaceModule) DepsMutator(context BottomUpMutatorContext) {
}

func (n *NamespaceModule) GenerateAndroidBuildActions(ctx ModuleContext) {
}

func (n *NamespaceModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func (n *NamespaceModule) Name() (name string) {
	return *n.nameProperties.Name
}

func NamespaceFactory() Module {
	module := &NamespaceModule{}

	name := "soong_namespace"
	module.nameProperties.Name = &name

	module.AddProperties(&module.properties)
	return module
}

func RegisterNamespaceMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("namespace_deps", namespaceMutator).Parallel()
}

func namespaceMutator(ctx BottomUpMutatorContext) {
	module, ok := ctx.Module().(*NamespaceModule)
	if ok {
		err := module.resolver.FindNamespaceImports(module.namespace)
		if err != nil {
			ctx.ModuleErrorf(err.Error())
		}

		module.resolver.chooseId(module.namespace)
	}
}
