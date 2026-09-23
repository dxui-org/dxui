package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon/catalog"
)

func TestGalleryRegistryCoversFoundationsAndPublicConstructors(t *testing.T) {
	want := []string{
		"Box", "Scroll", "VirtualList",
		"Colors", "Theme",
		"Text", "Icon", "Image", "Avatar", "Badge", "ProgressBar",
		"Button", "ButtonGroup", "InputGroup", "Input", "Textarea", "Select", "Tabs", "Menu", "Checkbox", "Radio", "ToggleSwitch", "Slider", "Popover", "Tooltip",
	}
	state := newGalleryState("Button")
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), state, makeAssets())
	if len(examples) != len(want) {
		t.Fatalf("registry length = %d, want %d", len(examples), len(want))
	}
	seen := make(map[string]bool, len(examples))
	for _, example := range examples {
		if seen[example.Name] {
			t.Errorf("duplicate component %q", example.Name)
		}
		seen[example.Name] = true
		if example.Category == "" || example.Purpose == "" || len(example.Coverage) == 0 {
			t.Errorf("component %q has incomplete navigation metadata", example.Name)
		}
		blocks := example.Build()
		if len(blocks) == 0 {
			t.Errorf("component %q has no examples", example.Name)
		}
		for index, block := range blocks {
			if block.Title == "" || block.Description == "" || block.Code == "" {
				t.Errorf("component %q block %d has incomplete explanatory content", example.Name, index)
			}
			if reflect.ValueOf(block.Preview).IsZero() {
				t.Errorf("component %q block %d has no live preview", example.Name, index)
			}
		}
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("public component %q is absent", name)
		}
	}
}

func TestGalleryRegistryMatchesExportedViewConstructors(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "view.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var constructors []string
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || !ast.IsExported(function.Name.Name) || function.Type.Results == nil {
			continue
		}
		if len(function.Type.Results.List) == 1 {
			if result, ok := function.Type.Results.List[0].Type.(*ast.Ident); ok && result.Name == "View" {
				if function.Name.Name == "Label" || function.Name.Name == "TextButton" {
					continue // convenience composition, not a distinct component kind
				}
				constructors = append(constructors, function.Name.Name)
			}
		}
	}
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), newGalleryState("Button"), makeAssets())
	registered := make([]string, 0, len(constructors))
	for _, example := range examples {
		if example.Category != "Foundations" {
			registered = append(registered, example.Name)
		}
	}
	sort.Strings(constructors)
	sort.Strings(registered)
	if !reflect.DeepEqual(registered, constructors) {
		t.Fatalf("registered component pages = %v, exported View constructors = %v", registered, constructors)
	}
}

func TestExampleSourcesUseOnlyRootDxuiAndOfficialIconPackages(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range parsed.Imports {
			pathValue, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			isDxuiSubpackage := strings.HasPrefix(pathValue, "github.com/dxui-org/dxui/")
			isOfficialIconPackage := pathValue == "github.com/dxui-org/dxui/icon" || pathValue == "github.com/dxui-org/dxui/icon/catalog"
			if (isDxuiSubpackage && !isOfficialIconPackage) || strings.Contains(pathValue, "go-sdl") {
				t.Errorf("%s imports forbidden package %q", path, pathValue)
			}
		}
	}
}

func TestControlledPagesExposeCurrentValues(t *testing.T) {
	state := newGalleryState("Input")
	for _, name := range []string{
		"Theme", "Icon", "Image", "Avatar", "Button", "Scroll", "Input", "Textarea",
		"Select", "Tabs", "Menu", "Checkbox", "Radio", "ToggleSwitch", "Slider", "Popover",
	} {
		if value, ok := currentValue(name, state); !ok || value == "" {
			t.Errorf("currentValue(%q) = %q, %t; want visible value", name, value, ok)
		}
	}
	for _, name := range []string{"Box", "Text", "Tooltip"} {
		if value, ok := currentValue(name, state); ok || value != "" {
			t.Errorf("currentValue(%q) = %q, %t; want inapplicable", name, value, ok)
		}
	}
}

func TestIconCatalogPaginationIncludesEveryName(t *testing.T) {
	seen := make(map[string]bool, len(catalog.Icons))
	for requestedPage := 0; ; requestedPage++ {
		entries, matches, page, pages := iconCatalogPage("", requestedPage)
		if matches != len(catalog.Icons) {
			t.Fatalf("unfiltered matches = %d, want %d", matches, len(catalog.Icons))
		}
		if page != requestedPage {
			t.Fatalf("page = %d, want requested page %d", page, requestedPage)
		}
		for _, entry := range entries {
			if seen[entry.Name] {
				t.Fatalf("duplicate paginated icon %q", entry.Name)
			}
			seen[entry.Name] = true
		}
		if requestedPage+1 == pages {
			break
		}
	}
	if len(seen) != len(catalog.Icons) {
		t.Fatalf("paginated icons = %d, want %d", len(seen), len(catalog.Icons))
	}
}

func TestIconCatalogSearchAndPageClamping(t *testing.T) {
	entries, matches, page, pages := iconCatalogPage("  ARROW-DOWN  ", 1000)
	if matches == 0 || len(entries) == 0 {
		t.Fatal("case-insensitive trimmed search returned no arrow-down matches")
	}
	if page != pages-1 {
		t.Fatalf("clamped page = %d, want final page %d", page, pages-1)
	}
	for _, entry := range entries {
		if !strings.Contains(entry.Name, "arrow-down") {
			t.Errorf("search result %q does not contain query", entry.Name)
		}
	}

	entries, matches, page, pages = iconCatalogPage("definitely-not-an-icon-name", 3)
	if entries != nil || matches != 0 || page != 0 || pages != 0 {
		t.Fatalf("empty search = (%v, %d, %d, %d), want (nil, 0, 0, 0)", entries, matches, page, pages)
	}
}

func TestGalleryRegistryKeepsCategoriesContiguous(t *testing.T) {
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), newGalleryState("Button"), makeAssets())
	closed := make(map[string]bool)
	lastCategory := ""
	for _, example := range examples {
		if example.Category == lastCategory {
			continue
		}
		if closed[example.Category] {
			t.Fatalf("category %q is split into multiple sidebar groups", example.Category)
		}
		if lastCategory != "" {
			closed[lastCategory] = true
		}
		lastCategory = example.Category
	}
}

func TestFoundationExamplesAreSeparateFromText(t *testing.T) {
	if got := len(textBlocks()); got != 3 {
		t.Fatalf("Text block count = %d, want 3", got)
	}
	if got := len(colorBlocks()); got != 1 {
		t.Fatalf("Colors block count = %d, want 1", got)
	}
	if got := len(themeBlocks(dxui.NewApp(dxui.AppOptions{}), newGalleryState("Theme"))); got != 3 {
		t.Fatalf("Theme block count = %d, want 3", got)
	}
}

func TestShowcaseThemeAppliesGlobalPaletteSettings(t *testing.T) {
	settings := galleryThemeSettings{primary: "violet", surface: "zinc", danger: "rose", success: "teal"}
	tests := []struct {
		name string
		dark bool
		want map[dxui.ColorToken]dxui.ColorValue
	}{
		{name: "light", want: map[dxui.ColorToken]dxui.ColorValue{
			dxui.Color.Semantic.Accent:      dxui.TokenColor(dxui.Color.Primitive.Violet600),
			dxui.Color.Semantic.AccentHover: dxui.TokenColor(dxui.Color.Primitive.Violet700),
			dxui.Color.Semantic.Surface:     dxui.TokenColor(dxui.Color.Primitive.Zinc50),
			dxui.Color.Semantic.SurfaceHigh: dxui.TokenColor(dxui.Color.Primitive.Zinc100),
			dxui.Color.Semantic.Text:        dxui.TokenColor(dxui.Color.Primitive.Zinc950),
			dxui.Color.Semantic.Border:      dxui.TokenColor(dxui.Color.Primitive.Zinc400),
			dxui.Color.Semantic.Danger:      dxui.TokenColor(dxui.Color.Primitive.Rose600),
			dxui.Color.Semantic.Success:     dxui.TokenColor(dxui.Color.Primitive.Teal600),
		}},
		{name: "dark", dark: true, want: map[dxui.ColorToken]dxui.ColorValue{
			dxui.Color.Semantic.Accent:      dxui.TokenColor(dxui.Color.Primitive.Violet400),
			dxui.Color.Semantic.AccentHover: dxui.TokenColor(dxui.Color.Primitive.Violet300),
			dxui.Color.Semantic.Surface:     dxui.TokenColor(dxui.Color.Primitive.Zinc950),
			dxui.Color.Semantic.SurfaceHigh: dxui.TokenColor(dxui.Color.Primitive.Zinc900),
			dxui.Color.Semantic.Text:        dxui.TokenColor(dxui.Color.Primitive.Zinc50),
			dxui.Color.Semantic.Border:      dxui.TokenColor(dxui.Color.Primitive.Zinc600),
			dxui.Color.Semantic.Danger:      dxui.TokenColor(dxui.Color.Primitive.Rose400),
			dxui.Color.Semantic.Success:     dxui.TokenColor(dxui.Color.Primitive.Teal400),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			theme := showcaseTheme(test.dark, settings)
			for token, want := range test.want {
				if got := theme.Semantic.Colors[token]; got != want {
					t.Errorf("semantic color %q = %#v, want %#v", token, got, want)
				}
			}
			if err := dxui.NewApp(dxui.AppOptions{}).SetTheme(theme); err != nil {
				t.Fatalf("customized theme rejected: %v", err)
			}
			component, ok := theme.Components[showcaseButtonTheme]
			if !ok {
				t.Fatal("custom showcase ComponentTheme is missing")
			}
			if want := dxui.Some(dxui.TokenColor(dxui.Color.Semantic.Danger)); component.Base.Background != want {
				t.Fatalf("custom component background = %#v, want %#v", component.Base.Background, want)
			}
		})
	}
}

func TestEveryPrimitiveFamilyCanConfigureGlobalThemeColors(t *testing.T) {
	if got := len(primitiveColorScales); got != 26 {
		t.Fatalf("Primitive color family count = %d, want 26", got)
	}
	for _, scale := range primitiveColorScales {
		t.Run(scale.name, func(t *testing.T) {
			settings := galleryThemeSettings{primary: scale.name, surface: scale.name, danger: scale.name, success: scale.name}
			light := showcaseTheme(false, settings)
			dark := showcaseTheme(true, settings)
			checks := []struct {
				name string
				got  dxui.ColorValue
				want dxui.ColorValue
			}{
				{name: "light primary", got: light.Semantic.Colors[dxui.Color.Semantic.Accent], want: dxui.TokenColor(scale.tokens[6])},
				{name: "light surface", got: light.Semantic.Colors[dxui.Color.Semantic.Surface], want: dxui.TokenColor(scale.tokens[0])},
				{name: "light danger", got: light.Semantic.Colors[dxui.Color.Semantic.Danger], want: dxui.TokenColor(scale.tokens[6])},
				{name: "light success", got: light.Semantic.Colors[dxui.Color.Semantic.Success], want: dxui.TokenColor(scale.tokens[6])},
				{name: "dark primary", got: dark.Semantic.Colors[dxui.Color.Semantic.Accent], want: dxui.TokenColor(scale.tokens[4])},
				{name: "dark surface", got: dark.Semantic.Colors[dxui.Color.Semantic.Surface], want: dxui.TokenColor(scale.tokens[10])},
				{name: "dark danger", got: dark.Semantic.Colors[dxui.Color.Semantic.Danger], want: dxui.TokenColor(scale.tokens[4])},
				{name: "dark success", got: dark.Semantic.Colors[dxui.Color.Semantic.Success], want: dxui.TokenColor(scale.tokens[4])},
			}
			for _, check := range checks {
				if check.got != check.want {
					t.Errorf("%s = %#v, want %#v", check.name, check.got, check.want)
				}
			}
		})
	}
}

func TestThemeGallerySnippetsInstallCompleteThemes(t *testing.T) {
	blocks := themeBlocks(dxui.NewApp(dxui.AppOptions{}), newGalleryState("Theme"))
	global := blocks[1].Code
	for _, fragment := range []string{
		"if dark {", "dxui.DarkTheme()", "Semantic.AccentHover", "Semantic.SurfaceHigh",
		"Semantic.Text", "Semantic.Border", "Semantic.Danger", "Semantic.Success", "app.SetTheme(theme)",
	} {
		if !strings.Contains(global, fragment) {
			t.Errorf("global theme snippet does not contain %q", fragment)
		}
	}
	component := blocks[2].Code
	for _, fragment := range []string{"theme := dxui.LightTheme()", "theme.Components[emphasis]", "app.SetTheme(theme)"} {
		if !strings.Contains(component, fragment) {
			t.Errorf("component theme snippet does not contain %q", fragment)
		}
	}
}

func TestThemeSwatchChoicesApplyEveryRole(t *testing.T) {
	app := dxui.NewApp(dxui.AppOptions{})
	state := newGalleryState("Theme")
	for _, test := range []struct {
		role  string
		value string
		got   func() string
	}{
		{role: "primary", value: "violet", got: func() string { return state.theme.primary }},
		{role: "surface", value: "zinc", got: func() string { return state.theme.surface }},
		{role: "danger", value: "rose", got: func() string { return state.theme.danger }},
		{role: "success", value: "teal", got: func() string { return state.theme.success }},
	} {
		setGalleryThemeChoice(app, state, test.role, test.value)
		if got := test.got(); got != test.value {
			t.Errorf("%s swatch set %q, want %q", test.role, got, test.value)
		}
		if !strings.Contains(state.feedback, titleCase(test.role)+" = "+titleCase(test.value)) {
			t.Errorf("%s feedback = %q", test.role, state.feedback)
		}
	}
}

func TestThemeSwatchesUseMappedLightAndDarkShades(t *testing.T) {
	scale, ok := primitiveColorScaleByName("violet")
	if !ok {
		t.Fatal("violet scale is missing")
	}
	tests := []struct {
		name                   string
		role                   string
		dark                   bool
		wantBackground, wantFG dxui.ColorToken
	}{
		{name: "light primary", role: "primary", wantBackground: scale.tokens[6], wantFG: scale.tokens[0]},
		{name: "dark primary", role: "primary", dark: true, wantBackground: scale.tokens[4], wantFG: scale.tokens[10]},
		{name: "light surface", role: "surface", wantBackground: scale.tokens[0], wantFG: scale.tokens[10]},
		{name: "dark surface", role: "surface", dark: true, wantBackground: scale.tokens[10], wantFG: scale.tokens[0]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			background, foreground := themeScaleSwatchColors(scale, test.role, test.dark)
			if background != test.wantBackground || foreground != test.wantFG {
				t.Fatalf("colors = (%q, %q), want (%q, %q)", background, foreground, test.wantBackground, test.wantFG)
			}
		})
	}
	if got := themeScaleSelectionBorder(scale, false); got != scale.tokens[10] {
		t.Errorf("light selection border = %q, want %q", got, scale.tokens[10])
	}
	if got := themeScaleSelectionBorder(scale, true); got != scale.tokens[0] {
		t.Errorf("dark selection border = %q, want %q", got, scale.tokens[0])
	}
}

func TestInitialComponentSelection(t *testing.T) {
	state := newGalleryState("unknown")
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), state, makeAssets())
	state.selectKnown(examples, "textarea")
	if state.selected != "Textarea" {
		t.Fatalf("case-insensitive selection = %q, want Textarea", state.selected)
	}
	state.selectKnown(examples, "unknown")
	if state.selected != examples[0].Name {
		t.Fatalf("unknown selection = %q, want fallback %q", state.selected, examples[0].Name)
	}
}

func TestDefaultComponentFlagSelectsBox(t *testing.T) {
	if defaultComponent != "Box" {
		t.Fatalf("default component = %q, want Box", defaultComponent)
	}
}

func TestLayoutPagesStartWithDirectionExamples(t *testing.T) {
	if got := verticalBoxBlocks()[0].Title; got != "Top-to-bottom layout" {
		t.Errorf("first Box example = %q, want top-to-bottom introduction", got)
	}
	if got := horizontalBoxBlocks()[0].Title; got != "Left-to-right layout" {
		t.Errorf("horizontal Box example = %q, want left-to-right introduction", got)
	}
}

func TestCommonBehaviorExamplesLiveWithSemanticControls(t *testing.T) {
	for _, block := range verticalBoxBlocks() {
		if strings.Contains(block.Title, "Key") || strings.Contains(block.Code, "PointerNone") {
			t.Fatalf("Box contains unrelated common-behavior block %q", block.Title)
		}
	}
	if blocks := inputBlocks(newGalleryState("Input")); !hasBlockTitle(blocks, "Keyed identity across reorder") {
		t.Error("Input page does not contain the keyed identity example")
	}
	if blocks := buttonBlocks(newGalleryState("Button"), makeAssets()); !strings.Contains(blocks[7].Code, "PointerNone") {
		t.Error("Button disabled comparison does not contain PointerNone")
	}
}

func hasBlockTitle(blocks []exampleBlock, title string) bool {
	for _, block := range blocks {
		if block.Title == title {
			return true
		}
	}
	return false
}

func TestButtonGalleryHasRequestedExamples(t *testing.T) {
	state := newGalleryState("Button")
	blocks := buttonBlocks(state, makeAssets())
	want := []string{
		"Default",
		"Button sizes",
		"Built-in button variants",
		"Soft buttons",
		"Outline buttons",
		"Dash buttons",
		"Ghost buttons and Link button",
		"Disabled buttons",
		"Square button and circle button",
		"Button with Icon",
		"Button with loading spinner",
		"Advanced customization",
	}
	if len(blocks) != len(want) {
		t.Fatalf("Button block count = %d, want %d", len(blocks), len(want))
	}
	for index := range want {
		if blocks[index].Title != want[index] {
			t.Errorf("Button block %d title = %q, want %q", index, blocks[index].Title, want[index])
		}
	}
	if strings.Contains(blocks[2].Code, "\t") {
		t.Fatal("button variants code contains a tab unsupported by the gallery text renderer")
	}
}

func TestButtonGroupGalleryHasRequestedExamples(t *testing.T) {
	state := newGalleryState("ButtonGroup")
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), state, makeAssets())
	var blocks []exampleBlock
	for _, example := range examples {
		if example.Name == "ButtonGroup" {
			blocks = example.Build()
			break
		}
	}
	want := []string{
		"Default ButtonGroup",
		"ButtonGroup with Dividers",
		"Vertical ButtonGroup",
		"Mixed Disabled Buttons",
	}
	if len(blocks) != len(want) {
		t.Fatalf("ButtonGroup block count = %d, want %d", len(blocks), len(want))
	}
	for index := range want {
		if blocks[index].Title != want[index] {
			t.Errorf("ButtonGroup block %d title = %q, want %q", index, blocks[index].Title, want[index])
		}
	}
}

func TestInputGroupGalleryHasRequestedExamples(t *testing.T) {
	examples := componentRegistry(dxui.NewApp(dxui.AppOptions{}), newGalleryState("InputGroup"), makeAssets())
	var blocks []exampleBlock
	for _, example := range examples {
		if example.Name == "InputGroup" {
			blocks = example.Build()
			break
		}
	}
	want := []string{"Search field", "Prefix icon", "Suffix unit", "Suffix action Button"}
	if len(blocks) != len(want) {
		t.Fatalf("InputGroup block count = %d, want %d", len(blocks), len(want))
	}
	for index := range want {
		if blocks[index].Title != want[index] {
			t.Errorf("InputGroup block %d title = %q, want %q", index, blocks[index].Title, want[index])
		}
	}
}

func TestSidebarNavigationReservesScrollbarGutter(t *testing.T) {
	const defaultScrollbarFootprint = 8 // 6-unit hover thumb plus the 2-unit right inset.
	if sidebarScrollbarGutter < defaultScrollbarFootprint {
		t.Fatalf("scrollbar gutter = %d, want at least %d", sidebarScrollbarGutter, defaultScrollbarFootprint)
	}
	if got, want := sidebarMenuWidth+sidebarScrollbarGutter, sidebarWidth-sidebarPadding-sidebarRightPadding; got != want {
		t.Fatalf("menu width plus gutter = %d, want inner sidebar width %d", got, want)
	}
	if menuGap := sidebarScrollbarGutter - defaultScrollbarFootprint; menuGap <= sidebarRightPadding {
		t.Fatalf("menu-to-scrollbar gap = %d, want greater than right padding %d", menuGap, sidebarRightPadding)
	}
}
