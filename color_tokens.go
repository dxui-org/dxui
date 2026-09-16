package dxui

// colorTokenNamespace provides discoverable, typed color tokens without
// exposing stringly-typed names to applications.
type colorTokenNamespace struct {
	Primitive primitiveColorTokens
	Semantic  semanticColorTokens
}

type primitiveColorTokens struct {
	White, Black                                                                                                                      ColorToken
	Red50, Red100, Red200, Red300, Red400, Red500, Red600, Red700, Red800, Red900, Red950                                             ColorToken
	Orange50, Orange100, Orange200, Orange300, Orange400, Orange500, Orange600, Orange700, Orange800, Orange900, Orange950            ColorToken
	Amber50, Amber100, Amber200, Amber300, Amber400, Amber500, Amber600, Amber700, Amber800, Amber900, Amber950                       ColorToken
	Yellow50, Yellow100, Yellow200, Yellow300, Yellow400, Yellow500, Yellow600, Yellow700, Yellow800, Yellow900, Yellow950            ColorToken
	Lime50, Lime100, Lime200, Lime300, Lime400, Lime500, Lime600, Lime700, Lime800, Lime900, Lime950                                  ColorToken
	Green50, Green100, Green200, Green300, Green400, Green500, Green600, Green700, Green800, Green900, Green950                       ColorToken
	Emerald50, Emerald100, Emerald200, Emerald300, Emerald400, Emerald500, Emerald600, Emerald700, Emerald800, Emerald900, Emerald950 ColorToken
	Teal50, Teal100, Teal200, Teal300, Teal400, Teal500, Teal600, Teal700, Teal800, Teal900, Teal950                                  ColorToken
	Cyan50, Cyan100, Cyan200, Cyan300, Cyan400, Cyan500, Cyan600, Cyan700, Cyan800, Cyan900, Cyan950                                  ColorToken
	Sky50, Sky100, Sky200, Sky300, Sky400, Sky500, Sky600, Sky700, Sky800, Sky900, Sky950                                             ColorToken
	Blue50, Blue100, Blue200, Blue300, Blue400, Blue500, Blue600, Blue700, Blue800, Blue900, Blue950                                  ColorToken
	Indigo50, Indigo100, Indigo200, Indigo300, Indigo400, Indigo500, Indigo600, Indigo700, Indigo800, Indigo900, Indigo950            ColorToken
	Violet50, Violet100, Violet200, Violet300, Violet400, Violet500, Violet600, Violet700, Violet800, Violet900, Violet950            ColorToken
	Purple50, Purple100, Purple200, Purple300, Purple400, Purple500, Purple600, Purple700, Purple800, Purple900, Purple950            ColorToken
	Fuchsia50, Fuchsia100, Fuchsia200, Fuchsia300, Fuchsia400, Fuchsia500, Fuchsia600, Fuchsia700, Fuchsia800, Fuchsia900, Fuchsia950 ColorToken
	Pink50, Pink100, Pink200, Pink300, Pink400, Pink500, Pink600, Pink700, Pink800, Pink900, Pink950                                  ColorToken
	Rose50, Rose100, Rose200, Rose300, Rose400, Rose500, Rose600, Rose700, Rose800, Rose900, Rose950                                  ColorToken
	Slate50, Slate100, Slate200, Slate300, Slate400, Slate500, Slate600, Slate700, Slate800, Slate900, Slate950                       ColorToken
	Gray50, Gray100, Gray200, Gray300, Gray400, Gray500, Gray600, Gray700, Gray800, Gray900, Gray950                                  ColorToken
	Zinc50, Zinc100, Zinc200, Zinc300, Zinc400, Zinc500, Zinc600, Zinc700, Zinc800, Zinc900, Zinc950                                  ColorToken
	Neutral50, Neutral100, Neutral200, Neutral300, Neutral400, Neutral500, Neutral600, Neutral700, Neutral800, Neutral900, Neutral950 ColorToken
	Stone50, Stone100, Stone200, Stone300, Stone400, Stone500, Stone600, Stone700, Stone800, Stone900, Stone950                       ColorToken
	Taupe50, Taupe100, Taupe200, Taupe300, Taupe400, Taupe500, Taupe600, Taupe700, Taupe800, Taupe900, Taupe950                       ColorToken
	Mauve50, Mauve100, Mauve200, Mauve300, Mauve400, Mauve500, Mauve600, Mauve700, Mauve800, Mauve900, Mauve950                       ColorToken
	Mist50, Mist100, Mist200, Mist300, Mist400, Mist500, Mist600, Mist700, Mist800, Mist900, Mist950                                  ColorToken
	Olive50, Olive100, Olive200, Olive300, Olive400, Olive500, Olive600, Olive700, Olive800, Olive900, Olive950                       ColorToken
}

type semanticColorTokens struct {
	Surface, SurfaceHigh, Text, Accent, AccentHover, OnAccent, FocusRing, Danger, Success, Info, Warn, Border, Shadow ColorToken
}

// Color is the public color-token namespace.
var Color = colorTokenNamespace{
	Primitive: primitiveColorTokens{
		White: ColorPrimitivePaletteWhite, Black: ColorPrimitivePaletteBlack,
		Red50: ColorPrimitiveRed50, Red100: ColorPrimitiveRed100, Red200: ColorPrimitiveRed200, Red300: ColorPrimitiveRed300, Red400: ColorPrimitiveRed400, Red500: ColorPrimitiveRed500, Red600: ColorPrimitiveRed600, Red700: ColorPrimitiveRed700, Red800: ColorPrimitiveRed800, Red900: ColorPrimitiveRed900, Red950: ColorPrimitiveRed950,
		Orange50: ColorPrimitiveOrange50, Orange100: ColorPrimitiveOrange100, Orange200: ColorPrimitiveOrange200, Orange300: ColorPrimitiveOrange300, Orange400: ColorPrimitiveOrange400, Orange500: ColorPrimitiveOrange500, Orange600: ColorPrimitiveOrange600, Orange700: ColorPrimitiveOrange700, Orange800: ColorPrimitiveOrange800, Orange900: ColorPrimitiveOrange900, Orange950: ColorPrimitiveOrange950,
		Amber50: ColorPrimitiveAmber50, Amber100: ColorPrimitiveAmber100, Amber200: ColorPrimitiveAmber200, Amber300: ColorPrimitiveAmber300, Amber400: ColorPrimitiveAmber400, Amber500: ColorPrimitiveAmber500, Amber600: ColorPrimitiveAmber600, Amber700: ColorPrimitiveAmber700, Amber800: ColorPrimitiveAmber800, Amber900: ColorPrimitiveAmber900, Amber950: ColorPrimitiveAmber950,
		Yellow50: ColorPrimitiveYellow50, Yellow100: ColorPrimitiveYellow100, Yellow200: ColorPrimitiveYellow200, Yellow300: ColorPrimitiveYellow300, Yellow400: ColorPrimitiveYellow400, Yellow500: ColorPrimitiveYellow500, Yellow600: ColorPrimitiveYellow600, Yellow700: ColorPrimitiveYellow700, Yellow800: ColorPrimitiveYellow800, Yellow900: ColorPrimitiveYellow900, Yellow950: ColorPrimitiveYellow950,
		Lime50: ColorPrimitiveLime50, Lime100: ColorPrimitiveLime100, Lime200: ColorPrimitiveLime200, Lime300: ColorPrimitiveLime300, Lime400: ColorPrimitiveLime400, Lime500: ColorPrimitiveLime500, Lime600: ColorPrimitiveLime600, Lime700: ColorPrimitiveLime700, Lime800: ColorPrimitiveLime800, Lime900: ColorPrimitiveLime900, Lime950: ColorPrimitiveLime950,
		Green50: ColorPrimitiveGreen50, Green100: ColorPrimitiveGreen100, Green200: ColorPrimitiveGreen200, Green300: ColorPrimitiveGreen300, Green400: ColorPrimitiveGreen400, Green500: ColorPrimitiveGreen500, Green600: ColorPrimitiveGreen600, Green700: ColorPrimitiveGreen700, Green800: ColorPrimitiveGreen800, Green900: ColorPrimitiveGreen900, Green950: ColorPrimitiveGreen950,
		Emerald50: ColorPrimitiveEmerald50, Emerald100: ColorPrimitiveEmerald100, Emerald200: ColorPrimitiveEmerald200, Emerald300: ColorPrimitiveEmerald300, Emerald400: ColorPrimitiveEmerald400, Emerald500: ColorPrimitiveEmerald500, Emerald600: ColorPrimitiveEmerald600, Emerald700: ColorPrimitiveEmerald700, Emerald800: ColorPrimitiveEmerald800, Emerald900: ColorPrimitiveEmerald900, Emerald950: ColorPrimitiveEmerald950,
		Teal50: ColorPrimitiveTeal50, Teal100: ColorPrimitiveTeal100, Teal200: ColorPrimitiveTeal200, Teal300: ColorPrimitiveTeal300, Teal400: ColorPrimitiveTeal400, Teal500: ColorPrimitiveTeal500, Teal600: ColorPrimitiveTeal600, Teal700: ColorPrimitiveTeal700, Teal800: ColorPrimitiveTeal800, Teal900: ColorPrimitiveTeal900, Teal950: ColorPrimitiveTeal950,
		Cyan50: ColorPrimitiveCyan50, Cyan100: ColorPrimitiveCyan100, Cyan200: ColorPrimitiveCyan200, Cyan300: ColorPrimitiveCyan300, Cyan400: ColorPrimitiveCyan400, Cyan500: ColorPrimitiveCyan500, Cyan600: ColorPrimitiveCyan600, Cyan700: ColorPrimitiveCyan700, Cyan800: ColorPrimitiveCyan800, Cyan900: ColorPrimitiveCyan900, Cyan950: ColorPrimitiveCyan950,
		Sky50: ColorPrimitiveSky50, Sky100: ColorPrimitiveSky100, Sky200: ColorPrimitiveSky200, Sky300: ColorPrimitiveSky300, Sky400: ColorPrimitiveSky400, Sky500: ColorPrimitiveSky500, Sky600: ColorPrimitiveSky600, Sky700: ColorPrimitiveSky700, Sky800: ColorPrimitiveSky800, Sky900: ColorPrimitiveSky900, Sky950: ColorPrimitiveSky950,
		Blue50: ColorPrimitiveBlue50, Blue100: ColorPrimitiveBlue100, Blue200: ColorPrimitiveBlue200, Blue300: ColorPrimitiveBlue300, Blue400: ColorPrimitiveBlue400, Blue500: ColorPrimitiveBlue500, Blue600: ColorPrimitiveBlue600, Blue700: ColorPrimitiveBlue700, Blue800: ColorPrimitiveBlue800, Blue900: ColorPrimitiveBlue900, Blue950: ColorPrimitiveBlue950,
		Indigo50: ColorPrimitiveIndigo50, Indigo100: ColorPrimitiveIndigo100, Indigo200: ColorPrimitiveIndigo200, Indigo300: ColorPrimitiveIndigo300, Indigo400: ColorPrimitiveIndigo400, Indigo500: ColorPrimitiveIndigo500, Indigo600: ColorPrimitiveIndigo600, Indigo700: ColorPrimitiveIndigo700, Indigo800: ColorPrimitiveIndigo800, Indigo900: ColorPrimitiveIndigo900, Indigo950: ColorPrimitiveIndigo950,
		Violet50: ColorPrimitiveViolet50, Violet100: ColorPrimitiveViolet100, Violet200: ColorPrimitiveViolet200, Violet300: ColorPrimitiveViolet300, Violet400: ColorPrimitiveViolet400, Violet500: ColorPrimitiveViolet500, Violet600: ColorPrimitiveViolet600, Violet700: ColorPrimitiveViolet700, Violet800: ColorPrimitiveViolet800, Violet900: ColorPrimitiveViolet900, Violet950: ColorPrimitiveViolet950,
		Purple50: ColorPrimitivePurple50, Purple100: ColorPrimitivePurple100, Purple200: ColorPrimitivePurple200, Purple300: ColorPrimitivePurple300, Purple400: ColorPrimitivePurple400, Purple500: ColorPrimitivePurple500, Purple600: ColorPrimitivePurple600, Purple700: ColorPrimitivePurple700, Purple800: ColorPrimitivePurple800, Purple900: ColorPrimitivePurple900, Purple950: ColorPrimitivePurple950,
		Fuchsia50: ColorPrimitiveFuchsia50, Fuchsia100: ColorPrimitiveFuchsia100, Fuchsia200: ColorPrimitiveFuchsia200, Fuchsia300: ColorPrimitiveFuchsia300, Fuchsia400: ColorPrimitiveFuchsia400, Fuchsia500: ColorPrimitiveFuchsia500, Fuchsia600: ColorPrimitiveFuchsia600, Fuchsia700: ColorPrimitiveFuchsia700, Fuchsia800: ColorPrimitiveFuchsia800, Fuchsia900: ColorPrimitiveFuchsia900, Fuchsia950: ColorPrimitiveFuchsia950,
		Pink50: ColorPrimitivePink50, Pink100: ColorPrimitivePink100, Pink200: ColorPrimitivePink200, Pink300: ColorPrimitivePink300, Pink400: ColorPrimitivePink400, Pink500: ColorPrimitivePink500, Pink600: ColorPrimitivePink600, Pink700: ColorPrimitivePink700, Pink800: ColorPrimitivePink800, Pink900: ColorPrimitivePink900, Pink950: ColorPrimitivePink950,
		Rose50: ColorPrimitiveRose50, Rose100: ColorPrimitiveRose100, Rose200: ColorPrimitiveRose200, Rose300: ColorPrimitiveRose300, Rose400: ColorPrimitiveRose400, Rose500: ColorPrimitiveRose500, Rose600: ColorPrimitiveRose600, Rose700: ColorPrimitiveRose700, Rose800: ColorPrimitiveRose800, Rose900: ColorPrimitiveRose900, Rose950: ColorPrimitiveRose950,
		Slate50: ColorPrimitiveSlate50, Slate100: ColorPrimitiveSlate100, Slate200: ColorPrimitiveSlate200, Slate300: ColorPrimitiveSlate300, Slate400: ColorPrimitiveSlate400, Slate500: ColorPrimitiveSlate500, Slate600: ColorPrimitiveSlate600, Slate700: ColorPrimitiveSlate700, Slate800: ColorPrimitiveSlate800, Slate900: ColorPrimitiveSlate900, Slate950: ColorPrimitiveSlate950,
		Gray50: ColorPrimitiveGray50, Gray100: ColorPrimitiveGray100, Gray200: ColorPrimitiveGray200, Gray300: ColorPrimitiveGray300, Gray400: ColorPrimitiveGray400, Gray500: ColorPrimitiveGray500, Gray600: ColorPrimitiveGray600, Gray700: ColorPrimitiveGray700, Gray800: ColorPrimitiveGray800, Gray900: ColorPrimitiveGray900, Gray950: ColorPrimitiveGray950,
		Zinc50: ColorPrimitiveZinc50, Zinc100: ColorPrimitiveZinc100, Zinc200: ColorPrimitiveZinc200, Zinc300: ColorPrimitiveZinc300, Zinc400: ColorPrimitiveZinc400, Zinc500: ColorPrimitiveZinc500, Zinc600: ColorPrimitiveZinc600, Zinc700: ColorPrimitiveZinc700, Zinc800: ColorPrimitiveZinc800, Zinc900: ColorPrimitiveZinc900, Zinc950: ColorPrimitiveZinc950,
		Neutral50: ColorPrimitiveNeutral50, Neutral100: ColorPrimitiveNeutral100, Neutral200: ColorPrimitiveNeutral200, Neutral300: ColorPrimitiveNeutral300, Neutral400: ColorPrimitiveNeutral400, Neutral500: ColorPrimitiveNeutral500, Neutral600: ColorPrimitiveNeutral600, Neutral700: ColorPrimitiveNeutral700, Neutral800: ColorPrimitiveNeutral800, Neutral900: ColorPrimitiveNeutral900, Neutral950: ColorPrimitiveNeutral950,
		Stone50: ColorPrimitiveStone50, Stone100: ColorPrimitiveStone100, Stone200: ColorPrimitiveStone200, Stone300: ColorPrimitiveStone300, Stone400: ColorPrimitiveStone400, Stone500: ColorPrimitiveStone500, Stone600: ColorPrimitiveStone600, Stone700: ColorPrimitiveStone700, Stone800: ColorPrimitiveStone800, Stone900: ColorPrimitiveStone900, Stone950: ColorPrimitiveStone950,
		Taupe50: ColorPrimitiveTaupe50, Taupe100: ColorPrimitiveTaupe100, Taupe200: ColorPrimitiveTaupe200, Taupe300: ColorPrimitiveTaupe300, Taupe400: ColorPrimitiveTaupe400, Taupe500: ColorPrimitiveTaupe500, Taupe600: ColorPrimitiveTaupe600, Taupe700: ColorPrimitiveTaupe700, Taupe800: ColorPrimitiveTaupe800, Taupe900: ColorPrimitiveTaupe900, Taupe950: ColorPrimitiveTaupe950,
		Mauve50: ColorPrimitiveMauve50, Mauve100: ColorPrimitiveMauve100, Mauve200: ColorPrimitiveMauve200, Mauve300: ColorPrimitiveMauve300, Mauve400: ColorPrimitiveMauve400, Mauve500: ColorPrimitiveMauve500, Mauve600: ColorPrimitiveMauve600, Mauve700: ColorPrimitiveMauve700, Mauve800: ColorPrimitiveMauve800, Mauve900: ColorPrimitiveMauve900, Mauve950: ColorPrimitiveMauve950,
		Mist50: ColorPrimitiveMist50, Mist100: ColorPrimitiveMist100, Mist200: ColorPrimitiveMist200, Mist300: ColorPrimitiveMist300, Mist400: ColorPrimitiveMist400, Mist500: ColorPrimitiveMist500, Mist600: ColorPrimitiveMist600, Mist700: ColorPrimitiveMist700, Mist800: ColorPrimitiveMist800, Mist900: ColorPrimitiveMist900, Mist950: ColorPrimitiveMist950,
		Olive50: ColorPrimitiveOlive50, Olive100: ColorPrimitiveOlive100, Olive200: ColorPrimitiveOlive200, Olive300: ColorPrimitiveOlive300, Olive400: ColorPrimitiveOlive400, Olive500: ColorPrimitiveOlive500, Olive600: ColorPrimitiveOlive600, Olive700: ColorPrimitiveOlive700, Olive800: ColorPrimitiveOlive800, Olive900: ColorPrimitiveOlive900, Olive950: ColorPrimitiveOlive950,
	},
	Semantic: semanticColorTokens{
		Surface: ColorSemanticSurface, SurfaceHigh: ColorSemanticSurfaceHi,
		Text: ColorSemanticText, Accent: ColorSemanticAccent,
		AccentHover: ColorSemanticAccentHover, Danger: ColorSemanticDanger,
		OnAccent: ColorSemanticOnAccent, FocusRing: ColorSemanticFocusRing,
		Success: ColorSemanticSuccess, Info: ColorSemanticInfo, Warn: ColorSemanticWarn,
		Border: ColorSemanticBorder, Shadow: ColorSemanticShadow,
	},
}

const (
	ColorPrimitivePaletteWhite ColorToken = "primitive.palette.white"
	ColorPrimitivePaletteBlack ColorToken = "primitive.palette.black"
	ColorPrimitiveRed50        ColorToken = "primitive.red.50"
	ColorPrimitiveRed100       ColorToken = "primitive.red.100"
	ColorPrimitiveRed200       ColorToken = "primitive.red.200"
	ColorPrimitiveRed300       ColorToken = "primitive.red.300"
	ColorPrimitiveRed400       ColorToken = "primitive.red.400"
	ColorPrimitiveRed500       ColorToken = "primitive.red.500"
	ColorPrimitiveRed600       ColorToken = "primitive.red.600"
	ColorPrimitiveRed700       ColorToken = "primitive.red.700"
	ColorPrimitiveRed800       ColorToken = "primitive.red.800"
	ColorPrimitiveRed900       ColorToken = "primitive.red.900"
	ColorPrimitiveRed950       ColorToken = "primitive.red.950"
	ColorPrimitiveOrange50     ColorToken = "primitive.orange.50"
	ColorPrimitiveOrange100    ColorToken = "primitive.orange.100"
	ColorPrimitiveOrange200    ColorToken = "primitive.orange.200"
	ColorPrimitiveOrange300    ColorToken = "primitive.orange.300"
	ColorPrimitiveOrange400    ColorToken = "primitive.orange.400"
	ColorPrimitiveOrange500    ColorToken = "primitive.orange.500"
	ColorPrimitiveOrange600    ColorToken = "primitive.orange.600"
	ColorPrimitiveOrange700    ColorToken = "primitive.orange.700"
	ColorPrimitiveOrange800    ColorToken = "primitive.orange.800"
	ColorPrimitiveOrange900    ColorToken = "primitive.orange.900"
	ColorPrimitiveOrange950    ColorToken = "primitive.orange.950"
	ColorPrimitiveAmber50      ColorToken = "primitive.amber.50"
	ColorPrimitiveAmber100     ColorToken = "primitive.amber.100"
	ColorPrimitiveAmber200     ColorToken = "primitive.amber.200"
	ColorPrimitiveAmber300     ColorToken = "primitive.amber.300"
	ColorPrimitiveAmber400     ColorToken = "primitive.amber.400"
	ColorPrimitiveAmber500     ColorToken = "primitive.amber.500"
	ColorPrimitiveAmber600     ColorToken = "primitive.amber.600"
	ColorPrimitiveAmber700     ColorToken = "primitive.amber.700"
	ColorPrimitiveAmber800     ColorToken = "primitive.amber.800"
	ColorPrimitiveAmber900     ColorToken = "primitive.amber.900"
	ColorPrimitiveAmber950     ColorToken = "primitive.amber.950"
	ColorPrimitiveYellow50     ColorToken = "primitive.yellow.50"
	ColorPrimitiveYellow100    ColorToken = "primitive.yellow.100"
	ColorPrimitiveYellow200    ColorToken = "primitive.yellow.200"
	ColorPrimitiveYellow300    ColorToken = "primitive.yellow.300"
	ColorPrimitiveYellow400    ColorToken = "primitive.yellow.400"
	ColorPrimitiveYellow500    ColorToken = "primitive.yellow.500"
	ColorPrimitiveYellow600    ColorToken = "primitive.yellow.600"
	ColorPrimitiveYellow700    ColorToken = "primitive.yellow.700"
	ColorPrimitiveYellow800    ColorToken = "primitive.yellow.800"
	ColorPrimitiveYellow900    ColorToken = "primitive.yellow.900"
	ColorPrimitiveYellow950    ColorToken = "primitive.yellow.950"
	ColorPrimitiveLime50       ColorToken = "primitive.lime.50"
	ColorPrimitiveLime100      ColorToken = "primitive.lime.100"
	ColorPrimitiveLime200      ColorToken = "primitive.lime.200"
	ColorPrimitiveLime300      ColorToken = "primitive.lime.300"
	ColorPrimitiveLime400      ColorToken = "primitive.lime.400"
	ColorPrimitiveLime500      ColorToken = "primitive.lime.500"
	ColorPrimitiveLime600      ColorToken = "primitive.lime.600"
	ColorPrimitiveLime700      ColorToken = "primitive.lime.700"
	ColorPrimitiveLime800      ColorToken = "primitive.lime.800"
	ColorPrimitiveLime900      ColorToken = "primitive.lime.900"
	ColorPrimitiveLime950      ColorToken = "primitive.lime.950"
	ColorPrimitiveGreen50      ColorToken = "primitive.green.50"
	ColorPrimitiveGreen100     ColorToken = "primitive.green.100"
	ColorPrimitiveGreen200     ColorToken = "primitive.green.200"
	ColorPrimitiveGreen300     ColorToken = "primitive.green.300"
	ColorPrimitiveGreen400     ColorToken = "primitive.green.400"
	ColorPrimitiveGreen500     ColorToken = "primitive.green.500"
	ColorPrimitiveGreen600     ColorToken = "primitive.green.600"
	ColorPrimitiveGreen700     ColorToken = "primitive.green.700"
	ColorPrimitiveGreen800     ColorToken = "primitive.green.800"
	ColorPrimitiveGreen900     ColorToken = "primitive.green.900"
	ColorPrimitiveGreen950     ColorToken = "primitive.green.950"
	ColorPrimitiveEmerald50    ColorToken = "primitive.emerald.50"
	ColorPrimitiveEmerald100   ColorToken = "primitive.emerald.100"
	ColorPrimitiveEmerald200   ColorToken = "primitive.emerald.200"
	ColorPrimitiveEmerald300   ColorToken = "primitive.emerald.300"
	ColorPrimitiveEmerald400   ColorToken = "primitive.emerald.400"
	ColorPrimitiveEmerald500   ColorToken = "primitive.emerald.500"
	ColorPrimitiveEmerald600   ColorToken = "primitive.emerald.600"
	ColorPrimitiveEmerald700   ColorToken = "primitive.emerald.700"
	ColorPrimitiveEmerald800   ColorToken = "primitive.emerald.800"
	ColorPrimitiveEmerald900   ColorToken = "primitive.emerald.900"
	ColorPrimitiveEmerald950   ColorToken = "primitive.emerald.950"
	ColorPrimitiveTeal50       ColorToken = "primitive.teal.50"
	ColorPrimitiveTeal100      ColorToken = "primitive.teal.100"
	ColorPrimitiveTeal200      ColorToken = "primitive.teal.200"
	ColorPrimitiveTeal300      ColorToken = "primitive.teal.300"
	ColorPrimitiveTeal400      ColorToken = "primitive.teal.400"
	ColorPrimitiveTeal500      ColorToken = "primitive.teal.500"
	ColorPrimitiveTeal600      ColorToken = "primitive.teal.600"
	ColorPrimitiveTeal700      ColorToken = "primitive.teal.700"
	ColorPrimitiveTeal800      ColorToken = "primitive.teal.800"
	ColorPrimitiveTeal900      ColorToken = "primitive.teal.900"
	ColorPrimitiveTeal950      ColorToken = "primitive.teal.950"
	ColorPrimitiveCyan50       ColorToken = "primitive.cyan.50"
	ColorPrimitiveCyan100      ColorToken = "primitive.cyan.100"
	ColorPrimitiveCyan200      ColorToken = "primitive.cyan.200"
	ColorPrimitiveCyan300      ColorToken = "primitive.cyan.300"
	ColorPrimitiveCyan400      ColorToken = "primitive.cyan.400"
	ColorPrimitiveCyan500      ColorToken = "primitive.cyan.500"
	ColorPrimitiveCyan600      ColorToken = "primitive.cyan.600"
	ColorPrimitiveCyan700      ColorToken = "primitive.cyan.700"
	ColorPrimitiveCyan800      ColorToken = "primitive.cyan.800"
	ColorPrimitiveCyan900      ColorToken = "primitive.cyan.900"
	ColorPrimitiveCyan950      ColorToken = "primitive.cyan.950"
	ColorPrimitiveSky50        ColorToken = "primitive.sky.50"
	ColorPrimitiveSky100       ColorToken = "primitive.sky.100"
	ColorPrimitiveSky200       ColorToken = "primitive.sky.200"
	ColorPrimitiveSky300       ColorToken = "primitive.sky.300"
	ColorPrimitiveSky400       ColorToken = "primitive.sky.400"
	ColorPrimitiveSky500       ColorToken = "primitive.sky.500"
	ColorPrimitiveSky600       ColorToken = "primitive.sky.600"
	ColorPrimitiveSky700       ColorToken = "primitive.sky.700"
	ColorPrimitiveSky800       ColorToken = "primitive.sky.800"
	ColorPrimitiveSky900       ColorToken = "primitive.sky.900"
	ColorPrimitiveSky950       ColorToken = "primitive.sky.950"
	ColorPrimitiveBlue50       ColorToken = "primitive.blue.50"
	ColorPrimitiveBlue100      ColorToken = "primitive.blue.100"
	ColorPrimitiveBlue200      ColorToken = "primitive.blue.200"
	ColorPrimitiveBlue300      ColorToken = "primitive.blue.300"
	ColorPrimitiveBlue400      ColorToken = "primitive.blue.400"
	ColorPrimitiveBlue500      ColorToken = "primitive.blue.500"
	ColorPrimitiveBlue600      ColorToken = "primitive.blue.600"
	ColorPrimitiveBlue700      ColorToken = "primitive.blue.700"
	ColorPrimitiveBlue800      ColorToken = "primitive.blue.800"
	ColorPrimitiveBlue900      ColorToken = "primitive.blue.900"
	ColorPrimitiveBlue950      ColorToken = "primitive.blue.950"
	ColorPrimitiveIndigo50     ColorToken = "primitive.indigo.50"
	ColorPrimitiveIndigo100    ColorToken = "primitive.indigo.100"
	ColorPrimitiveIndigo200    ColorToken = "primitive.indigo.200"
	ColorPrimitiveIndigo300    ColorToken = "primitive.indigo.300"
	ColorPrimitiveIndigo400    ColorToken = "primitive.indigo.400"
	ColorPrimitiveIndigo500    ColorToken = "primitive.indigo.500"
	ColorPrimitiveIndigo600    ColorToken = "primitive.indigo.600"
	ColorPrimitiveIndigo700    ColorToken = "primitive.indigo.700"
	ColorPrimitiveIndigo800    ColorToken = "primitive.indigo.800"
	ColorPrimitiveIndigo900    ColorToken = "primitive.indigo.900"
	ColorPrimitiveIndigo950    ColorToken = "primitive.indigo.950"
	ColorPrimitiveViolet50     ColorToken = "primitive.violet.50"
	ColorPrimitiveViolet100    ColorToken = "primitive.violet.100"
	ColorPrimitiveViolet200    ColorToken = "primitive.violet.200"
	ColorPrimitiveViolet300    ColorToken = "primitive.violet.300"
	ColorPrimitiveViolet400    ColorToken = "primitive.violet.400"
	ColorPrimitiveViolet500    ColorToken = "primitive.violet.500"
	ColorPrimitiveViolet600    ColorToken = "primitive.violet.600"
	ColorPrimitiveViolet700    ColorToken = "primitive.violet.700"
	ColorPrimitiveViolet800    ColorToken = "primitive.violet.800"
	ColorPrimitiveViolet900    ColorToken = "primitive.violet.900"
	ColorPrimitiveViolet950    ColorToken = "primitive.violet.950"
	ColorPrimitivePurple50     ColorToken = "primitive.purple.50"
	ColorPrimitivePurple100    ColorToken = "primitive.purple.100"
	ColorPrimitivePurple200    ColorToken = "primitive.purple.200"
	ColorPrimitivePurple300    ColorToken = "primitive.purple.300"
	ColorPrimitivePurple400    ColorToken = "primitive.purple.400"
	ColorPrimitivePurple500    ColorToken = "primitive.purple.500"
	ColorPrimitivePurple600    ColorToken = "primitive.purple.600"
	ColorPrimitivePurple700    ColorToken = "primitive.purple.700"
	ColorPrimitivePurple800    ColorToken = "primitive.purple.800"
	ColorPrimitivePurple900    ColorToken = "primitive.purple.900"
	ColorPrimitivePurple950    ColorToken = "primitive.purple.950"
	ColorPrimitiveFuchsia50    ColorToken = "primitive.fuchsia.50"
	ColorPrimitiveFuchsia100   ColorToken = "primitive.fuchsia.100"
	ColorPrimitiveFuchsia200   ColorToken = "primitive.fuchsia.200"
	ColorPrimitiveFuchsia300   ColorToken = "primitive.fuchsia.300"
	ColorPrimitiveFuchsia400   ColorToken = "primitive.fuchsia.400"
	ColorPrimitiveFuchsia500   ColorToken = "primitive.fuchsia.500"
	ColorPrimitiveFuchsia600   ColorToken = "primitive.fuchsia.600"
	ColorPrimitiveFuchsia700   ColorToken = "primitive.fuchsia.700"
	ColorPrimitiveFuchsia800   ColorToken = "primitive.fuchsia.800"
	ColorPrimitiveFuchsia900   ColorToken = "primitive.fuchsia.900"
	ColorPrimitiveFuchsia950   ColorToken = "primitive.fuchsia.950"
	ColorPrimitivePink50       ColorToken = "primitive.pink.50"
	ColorPrimitivePink100      ColorToken = "primitive.pink.100"
	ColorPrimitivePink200      ColorToken = "primitive.pink.200"
	ColorPrimitivePink300      ColorToken = "primitive.pink.300"
	ColorPrimitivePink400      ColorToken = "primitive.pink.400"
	ColorPrimitivePink500      ColorToken = "primitive.pink.500"
	ColorPrimitivePink600      ColorToken = "primitive.pink.600"
	ColorPrimitivePink700      ColorToken = "primitive.pink.700"
	ColorPrimitivePink800      ColorToken = "primitive.pink.800"
	ColorPrimitivePink900      ColorToken = "primitive.pink.900"
	ColorPrimitivePink950      ColorToken = "primitive.pink.950"
	ColorPrimitiveRose50       ColorToken = "primitive.rose.50"
	ColorPrimitiveRose100      ColorToken = "primitive.rose.100"
	ColorPrimitiveRose200      ColorToken = "primitive.rose.200"
	ColorPrimitiveRose300      ColorToken = "primitive.rose.300"
	ColorPrimitiveRose400      ColorToken = "primitive.rose.400"
	ColorPrimitiveRose500      ColorToken = "primitive.rose.500"
	ColorPrimitiveRose600      ColorToken = "primitive.rose.600"
	ColorPrimitiveRose700      ColorToken = "primitive.rose.700"
	ColorPrimitiveRose800      ColorToken = "primitive.rose.800"
	ColorPrimitiveRose900      ColorToken = "primitive.rose.900"
	ColorPrimitiveRose950      ColorToken = "primitive.rose.950"
	ColorPrimitiveSlate50      ColorToken = "primitive.slate.50"
	ColorPrimitiveSlate100     ColorToken = "primitive.slate.100"
	ColorPrimitiveSlate200     ColorToken = "primitive.slate.200"
	ColorPrimitiveSlate300     ColorToken = "primitive.slate.300"
	ColorPrimitiveSlate400     ColorToken = "primitive.slate.400"
	ColorPrimitiveSlate500     ColorToken = "primitive.slate.500"
	ColorPrimitiveSlate600     ColorToken = "primitive.slate.600"
	ColorPrimitiveSlate700     ColorToken = "primitive.slate.700"
	ColorPrimitiveSlate800     ColorToken = "primitive.slate.800"
	ColorPrimitiveSlate900     ColorToken = "primitive.slate.900"
	ColorPrimitiveSlate950     ColorToken = "primitive.slate.950"
	ColorPrimitiveGray50       ColorToken = "primitive.gray.50"
	ColorPrimitiveGray100      ColorToken = "primitive.gray.100"
	ColorPrimitiveGray200      ColorToken = "primitive.gray.200"
	ColorPrimitiveGray300      ColorToken = "primitive.gray.300"
	ColorPrimitiveGray400      ColorToken = "primitive.gray.400"
	ColorPrimitiveGray500      ColorToken = "primitive.gray.500"
	ColorPrimitiveGray600      ColorToken = "primitive.gray.600"
	ColorPrimitiveGray700      ColorToken = "primitive.gray.700"
	ColorPrimitiveGray800      ColorToken = "primitive.gray.800"
	ColorPrimitiveGray900      ColorToken = "primitive.gray.900"
	ColorPrimitiveGray950      ColorToken = "primitive.gray.950"
	ColorPrimitiveZinc50       ColorToken = "primitive.zinc.50"
	ColorPrimitiveZinc100      ColorToken = "primitive.zinc.100"
	ColorPrimitiveZinc200      ColorToken = "primitive.zinc.200"
	ColorPrimitiveZinc300      ColorToken = "primitive.zinc.300"
	ColorPrimitiveZinc400      ColorToken = "primitive.zinc.400"
	ColorPrimitiveZinc500      ColorToken = "primitive.zinc.500"
	ColorPrimitiveZinc600      ColorToken = "primitive.zinc.600"
	ColorPrimitiveZinc700      ColorToken = "primitive.zinc.700"
	ColorPrimitiveZinc800      ColorToken = "primitive.zinc.800"
	ColorPrimitiveZinc900      ColorToken = "primitive.zinc.900"
	ColorPrimitiveZinc950      ColorToken = "primitive.zinc.950"
	ColorPrimitiveNeutral50    ColorToken = "primitive.neutral.50"
	ColorPrimitiveNeutral100   ColorToken = "primitive.neutral.100"
	ColorPrimitiveNeutral200   ColorToken = "primitive.neutral.200"
	ColorPrimitiveNeutral300   ColorToken = "primitive.neutral.300"
	ColorPrimitiveNeutral400   ColorToken = "primitive.neutral.400"
	ColorPrimitiveNeutral500   ColorToken = "primitive.neutral.500"
	ColorPrimitiveNeutral600   ColorToken = "primitive.neutral.600"
	ColorPrimitiveNeutral700   ColorToken = "primitive.neutral.700"
	ColorPrimitiveNeutral800   ColorToken = "primitive.neutral.800"
	ColorPrimitiveNeutral900   ColorToken = "primitive.neutral.900"
	ColorPrimitiveNeutral950   ColorToken = "primitive.neutral.950"
	ColorPrimitiveStone50      ColorToken = "primitive.stone.50"
	ColorPrimitiveStone100     ColorToken = "primitive.stone.100"
	ColorPrimitiveStone200     ColorToken = "primitive.stone.200"
	ColorPrimitiveStone300     ColorToken = "primitive.stone.300"
	ColorPrimitiveStone400     ColorToken = "primitive.stone.400"
	ColorPrimitiveStone500     ColorToken = "primitive.stone.500"
	ColorPrimitiveStone600     ColorToken = "primitive.stone.600"
	ColorPrimitiveStone700     ColorToken = "primitive.stone.700"
	ColorPrimitiveStone800     ColorToken = "primitive.stone.800"
	ColorPrimitiveStone900     ColorToken = "primitive.stone.900"
	ColorPrimitiveStone950     ColorToken = "primitive.stone.950"
	ColorPrimitiveTaupe50      ColorToken = "primitive.taupe.50"
	ColorPrimitiveTaupe100     ColorToken = "primitive.taupe.100"
	ColorPrimitiveTaupe200     ColorToken = "primitive.taupe.200"
	ColorPrimitiveTaupe300     ColorToken = "primitive.taupe.300"
	ColorPrimitiveTaupe400     ColorToken = "primitive.taupe.400"
	ColorPrimitiveTaupe500     ColorToken = "primitive.taupe.500"
	ColorPrimitiveTaupe600     ColorToken = "primitive.taupe.600"
	ColorPrimitiveTaupe700     ColorToken = "primitive.taupe.700"
	ColorPrimitiveTaupe800     ColorToken = "primitive.taupe.800"
	ColorPrimitiveTaupe900     ColorToken = "primitive.taupe.900"
	ColorPrimitiveTaupe950     ColorToken = "primitive.taupe.950"
	ColorPrimitiveMauve50      ColorToken = "primitive.mauve.50"
	ColorPrimitiveMauve100     ColorToken = "primitive.mauve.100"
	ColorPrimitiveMauve200     ColorToken = "primitive.mauve.200"
	ColorPrimitiveMauve300     ColorToken = "primitive.mauve.300"
	ColorPrimitiveMauve400     ColorToken = "primitive.mauve.400"
	ColorPrimitiveMauve500     ColorToken = "primitive.mauve.500"
	ColorPrimitiveMauve600     ColorToken = "primitive.mauve.600"
	ColorPrimitiveMauve700     ColorToken = "primitive.mauve.700"
	ColorPrimitiveMauve800     ColorToken = "primitive.mauve.800"
	ColorPrimitiveMauve900     ColorToken = "primitive.mauve.900"
	ColorPrimitiveMauve950     ColorToken = "primitive.mauve.950"
	ColorPrimitiveMist50       ColorToken = "primitive.mist.50"
	ColorPrimitiveMist100      ColorToken = "primitive.mist.100"
	ColorPrimitiveMist200      ColorToken = "primitive.mist.200"
	ColorPrimitiveMist300      ColorToken = "primitive.mist.300"
	ColorPrimitiveMist400      ColorToken = "primitive.mist.400"
	ColorPrimitiveMist500      ColorToken = "primitive.mist.500"
	ColorPrimitiveMist600      ColorToken = "primitive.mist.600"
	ColorPrimitiveMist700      ColorToken = "primitive.mist.700"
	ColorPrimitiveMist800      ColorToken = "primitive.mist.800"
	ColorPrimitiveMist900      ColorToken = "primitive.mist.900"
	ColorPrimitiveMist950      ColorToken = "primitive.mist.950"
	ColorPrimitiveOlive50      ColorToken = "primitive.olive.50"
	ColorPrimitiveOlive100     ColorToken = "primitive.olive.100"
	ColorPrimitiveOlive200     ColorToken = "primitive.olive.200"
	ColorPrimitiveOlive300     ColorToken = "primitive.olive.300"
	ColorPrimitiveOlive400     ColorToken = "primitive.olive.400"
	ColorPrimitiveOlive500     ColorToken = "primitive.olive.500"
	ColorPrimitiveOlive600     ColorToken = "primitive.olive.600"
	ColorPrimitiveOlive700     ColorToken = "primitive.olive.700"
	ColorPrimitiveOlive800     ColorToken = "primitive.olive.800"
	ColorPrimitiveOlive900     ColorToken = "primitive.olive.900"
	ColorPrimitiveOlive950     ColorToken = "primitive.olive.950"
)

func builtinPrimitiveColors() map[ColorToken]RGBAColor {
	colors := make(map[ColorToken]RGBAColor, 292)
	colors[Color.Primitive.White] = RGBA(255, 255, 255, 255)
	colors[Color.Primitive.Black] = RGBA(0, 0, 0, 255)
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Red50, Color.Primitive.Red100, Color.Primitive.Red200, Color.Primitive.Red300, Color.Primitive.Red400, Color.Primitive.Red500, Color.Primitive.Red600, Color.Primitive.Red700, Color.Primitive.Red800, Color.Primitive.Red900, Color.Primitive.Red950}, [11]uint32{0xfef2f2, 0xfee2e2, 0xfecaca, 0xfca5a5, 0xf87171, 0xef4444, 0xdc2626, 0xb91c1c, 0x991b1b, 0x7f1d1d, 0x450a0a})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Orange50, Color.Primitive.Orange100, Color.Primitive.Orange200, Color.Primitive.Orange300, Color.Primitive.Orange400, Color.Primitive.Orange500, Color.Primitive.Orange600, Color.Primitive.Orange700, Color.Primitive.Orange800, Color.Primitive.Orange900, Color.Primitive.Orange950}, [11]uint32{0xfff7ed, 0xffedd5, 0xfed7aa, 0xfdba74, 0xfb923c, 0xf97316, 0xea580c, 0xc2410c, 0x9a3412, 0x7c2d12, 0x431407})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Amber50, Color.Primitive.Amber100, Color.Primitive.Amber200, Color.Primitive.Amber300, Color.Primitive.Amber400, Color.Primitive.Amber500, Color.Primitive.Amber600, Color.Primitive.Amber700, Color.Primitive.Amber800, Color.Primitive.Amber900, Color.Primitive.Amber950}, [11]uint32{0xfffbeb, 0xfef3c7, 0xfde68a, 0xfcd34d, 0xfbbf24, 0xf59e0b, 0xd97706, 0xb45309, 0x92400e, 0x78350f, 0x451a03})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Yellow50, Color.Primitive.Yellow100, Color.Primitive.Yellow200, Color.Primitive.Yellow300, Color.Primitive.Yellow400, Color.Primitive.Yellow500, Color.Primitive.Yellow600, Color.Primitive.Yellow700, Color.Primitive.Yellow800, Color.Primitive.Yellow900, Color.Primitive.Yellow950}, [11]uint32{0xfefce8, 0xfef9c3, 0xfef08a, 0xfde047, 0xfacc15, 0xeab308, 0xca8a04, 0xa16207, 0x854d0e, 0x713f12, 0x422006})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Lime50, Color.Primitive.Lime100, Color.Primitive.Lime200, Color.Primitive.Lime300, Color.Primitive.Lime400, Color.Primitive.Lime500, Color.Primitive.Lime600, Color.Primitive.Lime700, Color.Primitive.Lime800, Color.Primitive.Lime900, Color.Primitive.Lime950}, [11]uint32{0xf7fee7, 0xecfccb, 0xd9f99d, 0xbef264, 0xa3e635, 0x84cc16, 0x65a30d, 0x4d7c0f, 0x3f6212, 0x365314, 0x1a2e05})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Green50, Color.Primitive.Green100, Color.Primitive.Green200, Color.Primitive.Green300, Color.Primitive.Green400, Color.Primitive.Green500, Color.Primitive.Green600, Color.Primitive.Green700, Color.Primitive.Green800, Color.Primitive.Green900, Color.Primitive.Green950}, [11]uint32{0xf0fdf4, 0xdcfce7, 0xbbf7d0, 0x86efac, 0x4ade80, 0x22c55e, 0x16a34a, 0x15803d, 0x166534, 0x14532d, 0x052e16})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Emerald50, Color.Primitive.Emerald100, Color.Primitive.Emerald200, Color.Primitive.Emerald300, Color.Primitive.Emerald400, Color.Primitive.Emerald500, Color.Primitive.Emerald600, Color.Primitive.Emerald700, Color.Primitive.Emerald800, Color.Primitive.Emerald900, Color.Primitive.Emerald950}, [11]uint32{0xecfdf5, 0xd1fae5, 0xa7f3d0, 0x6ee7b7, 0x34d399, 0x10b981, 0x059669, 0x047857, 0x065f46, 0x064e3b, 0x022c22})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Teal50, Color.Primitive.Teal100, Color.Primitive.Teal200, Color.Primitive.Teal300, Color.Primitive.Teal400, Color.Primitive.Teal500, Color.Primitive.Teal600, Color.Primitive.Teal700, Color.Primitive.Teal800, Color.Primitive.Teal900, Color.Primitive.Teal950}, [11]uint32{0xf0fdfa, 0xccfbf1, 0x99f6e4, 0x5eead4, 0x2dd4bf, 0x14b8a6, 0x0d9488, 0x0f766e, 0x115e59, 0x134e4a, 0x042f2e})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Cyan50, Color.Primitive.Cyan100, Color.Primitive.Cyan200, Color.Primitive.Cyan300, Color.Primitive.Cyan400, Color.Primitive.Cyan500, Color.Primitive.Cyan600, Color.Primitive.Cyan700, Color.Primitive.Cyan800, Color.Primitive.Cyan900, Color.Primitive.Cyan950}, [11]uint32{0xecfeff, 0xcffafe, 0xa5f3fc, 0x67e8f9, 0x22d3ee, 0x06b6d4, 0x0891b2, 0x0e7490, 0x155e75, 0x164e63, 0x083344})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Sky50, Color.Primitive.Sky100, Color.Primitive.Sky200, Color.Primitive.Sky300, Color.Primitive.Sky400, Color.Primitive.Sky500, Color.Primitive.Sky600, Color.Primitive.Sky700, Color.Primitive.Sky800, Color.Primitive.Sky900, Color.Primitive.Sky950}, [11]uint32{0xf0f9ff, 0xe0f2fe, 0xbae6fd, 0x7dd3fc, 0x38bdf8, 0x0ea5e9, 0x0284c7, 0x0369a1, 0x075985, 0x0c4a6e, 0x082f49})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Blue50, Color.Primitive.Blue100, Color.Primitive.Blue200, Color.Primitive.Blue300, Color.Primitive.Blue400, Color.Primitive.Blue500, Color.Primitive.Blue600, Color.Primitive.Blue700, Color.Primitive.Blue800, Color.Primitive.Blue900, Color.Primitive.Blue950}, [11]uint32{0xeff6ff, 0xdbeafe, 0xbfdbfe, 0x93c5fd, 0x60a5fa, 0x3b82f6, 0x2563eb, 0x1d4ed8, 0x1e40af, 0x1e3a8a, 0x172554})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Indigo50, Color.Primitive.Indigo100, Color.Primitive.Indigo200, Color.Primitive.Indigo300, Color.Primitive.Indigo400, Color.Primitive.Indigo500, Color.Primitive.Indigo600, Color.Primitive.Indigo700, Color.Primitive.Indigo800, Color.Primitive.Indigo900, Color.Primitive.Indigo950}, [11]uint32{0xeef2ff, 0xe0e7ff, 0xc7d2fe, 0xa5b4fc, 0x818cf8, 0x6366f1, 0x4f46e5, 0x4338ca, 0x3730a3, 0x312e81, 0x1e1b4b})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Violet50, Color.Primitive.Violet100, Color.Primitive.Violet200, Color.Primitive.Violet300, Color.Primitive.Violet400, Color.Primitive.Violet500, Color.Primitive.Violet600, Color.Primitive.Violet700, Color.Primitive.Violet800, Color.Primitive.Violet900, Color.Primitive.Violet950}, [11]uint32{0xf5f3ff, 0xede9fe, 0xddd6fe, 0xc4b5fd, 0xa78bfa, 0x8b5cf6, 0x7c3aed, 0x6d28d9, 0x5b21b6, 0x4c1d95, 0x2e1065})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Purple50, Color.Primitive.Purple100, Color.Primitive.Purple200, Color.Primitive.Purple300, Color.Primitive.Purple400, Color.Primitive.Purple500, Color.Primitive.Purple600, Color.Primitive.Purple700, Color.Primitive.Purple800, Color.Primitive.Purple900, Color.Primitive.Purple950}, [11]uint32{0xfaf5ff, 0xf3e8ff, 0xe9d5ff, 0xd8b4fe, 0xc084fc, 0xa855f7, 0x9333ea, 0x7e22ce, 0x6b21a8, 0x581c87, 0x3b0764})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Fuchsia50, Color.Primitive.Fuchsia100, Color.Primitive.Fuchsia200, Color.Primitive.Fuchsia300, Color.Primitive.Fuchsia400, Color.Primitive.Fuchsia500, Color.Primitive.Fuchsia600, Color.Primitive.Fuchsia700, Color.Primitive.Fuchsia800, Color.Primitive.Fuchsia900, Color.Primitive.Fuchsia950}, [11]uint32{0xfdf4ff, 0xfae8ff, 0xf5d0fe, 0xf0abfc, 0xe879f9, 0xd946ef, 0xc026d3, 0xa21caf, 0x86198f, 0x701a75, 0x4a044e})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Pink50, Color.Primitive.Pink100, Color.Primitive.Pink200, Color.Primitive.Pink300, Color.Primitive.Pink400, Color.Primitive.Pink500, Color.Primitive.Pink600, Color.Primitive.Pink700, Color.Primitive.Pink800, Color.Primitive.Pink900, Color.Primitive.Pink950}, [11]uint32{0xfdf2f8, 0xfce7f3, 0xfbcfe8, 0xf9a8d4, 0xf472b6, 0xec4899, 0xdb2777, 0xbe185d, 0x9d174d, 0x831843, 0x500724})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Rose50, Color.Primitive.Rose100, Color.Primitive.Rose200, Color.Primitive.Rose300, Color.Primitive.Rose400, Color.Primitive.Rose500, Color.Primitive.Rose600, Color.Primitive.Rose700, Color.Primitive.Rose800, Color.Primitive.Rose900, Color.Primitive.Rose950}, [11]uint32{0xfff1f2, 0xffe4e6, 0xfecdd3, 0xfda4af, 0xfb7185, 0xf43f5e, 0xe11d48, 0xbe123c, 0x9f1239, 0x881337, 0x4c0519})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Slate50, Color.Primitive.Slate100, Color.Primitive.Slate200, Color.Primitive.Slate300, Color.Primitive.Slate400, Color.Primitive.Slate500, Color.Primitive.Slate600, Color.Primitive.Slate700, Color.Primitive.Slate800, Color.Primitive.Slate900, Color.Primitive.Slate950}, [11]uint32{0xf8fafc, 0xf1f5f9, 0xe2e8f0, 0xcbd5e1, 0x94a3b8, 0x64748b, 0x475569, 0x334155, 0x1e293b, 0x0f172a, 0x020617})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Gray50, Color.Primitive.Gray100, Color.Primitive.Gray200, Color.Primitive.Gray300, Color.Primitive.Gray400, Color.Primitive.Gray500, Color.Primitive.Gray600, Color.Primitive.Gray700, Color.Primitive.Gray800, Color.Primitive.Gray900, Color.Primitive.Gray950}, [11]uint32{0xf9fafb, 0xf3f4f6, 0xe5e7eb, 0xd1d5db, 0x9ca3af, 0x6b7280, 0x4b5563, 0x374151, 0x1f2937, 0x111827, 0x030712})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Zinc50, Color.Primitive.Zinc100, Color.Primitive.Zinc200, Color.Primitive.Zinc300, Color.Primitive.Zinc400, Color.Primitive.Zinc500, Color.Primitive.Zinc600, Color.Primitive.Zinc700, Color.Primitive.Zinc800, Color.Primitive.Zinc900, Color.Primitive.Zinc950}, [11]uint32{0xfafafa, 0xf4f4f5, 0xe4e4e7, 0xd4d4d8, 0xa1a1aa, 0x71717a, 0x52525b, 0x3f3f46, 0x27272a, 0x18181b, 0x09090b})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Neutral50, Color.Primitive.Neutral100, Color.Primitive.Neutral200, Color.Primitive.Neutral300, Color.Primitive.Neutral400, Color.Primitive.Neutral500, Color.Primitive.Neutral600, Color.Primitive.Neutral700, Color.Primitive.Neutral800, Color.Primitive.Neutral900, Color.Primitive.Neutral950}, [11]uint32{0xfafafa, 0xf5f5f5, 0xe5e5e5, 0xd4d4d4, 0xa3a3a3, 0x737373, 0x525252, 0x404040, 0x262626, 0x171717, 0x0a0a0a})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Stone50, Color.Primitive.Stone100, Color.Primitive.Stone200, Color.Primitive.Stone300, Color.Primitive.Stone400, Color.Primitive.Stone500, Color.Primitive.Stone600, Color.Primitive.Stone700, Color.Primitive.Stone800, Color.Primitive.Stone900, Color.Primitive.Stone950}, [11]uint32{0xfafaf9, 0xf5f5f4, 0xe7e5e4, 0xd6d3d1, 0xa8a29e, 0x78716c, 0x57534e, 0x44403c, 0x292524, 0x1c1917, 0x0c0a09})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Taupe50, Color.Primitive.Taupe100, Color.Primitive.Taupe200, Color.Primitive.Taupe300, Color.Primitive.Taupe400, Color.Primitive.Taupe500, Color.Primitive.Taupe600, Color.Primitive.Taupe700, Color.Primitive.Taupe800, Color.Primitive.Taupe900, Color.Primitive.Taupe950}, [11]uint32{0xfbfaf9, 0xf3f1f1, 0xe8e4e3, 0xd8d2d0, 0xaba09c, 0x7c6d67, 0x5b4f4b, 0x473c39, 0x2b2422, 0x1d1816, 0x0c0a09})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Mauve50, Color.Primitive.Mauve100, Color.Primitive.Mauve200, Color.Primitive.Mauve300, Color.Primitive.Mauve400, Color.Primitive.Mauve500, Color.Primitive.Mauve600, Color.Primitive.Mauve700, Color.Primitive.Mauve800, Color.Primitive.Mauve900, Color.Primitive.Mauve950}, [11]uint32{0xfafafa, 0xf3f1f3, 0xe7e4e7, 0xd7d0d7, 0xa89ea9, 0x79697b, 0x594c5b, 0x463947, 0x2a212c, 0x1d161e, 0x0c090c})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Mist50, Color.Primitive.Mist100, Color.Primitive.Mist200, Color.Primitive.Mist300, Color.Primitive.Mist400, Color.Primitive.Mist500, Color.Primitive.Mist600, Color.Primitive.Mist700, Color.Primitive.Mist800, Color.Primitive.Mist900, Color.Primitive.Mist950}, [11]uint32{0xf9fbfb, 0xf1f3f3, 0xe3e7e8, 0xd0d6d8, 0x9ca8ab, 0x67787c, 0x4b585b, 0x394447, 0x22292b, 0x161b1d, 0x090b0c})
	addPrimitiveScale(colors, [11]ColorToken{Color.Primitive.Olive50, Color.Primitive.Olive100, Color.Primitive.Olive200, Color.Primitive.Olive300, Color.Primitive.Olive400, Color.Primitive.Olive500, Color.Primitive.Olive600, Color.Primitive.Olive700, Color.Primitive.Olive800, Color.Primitive.Olive900, Color.Primitive.Olive950}, [11]uint32{0xfbfbf9, 0xf4f4f0, 0xe8e8e3, 0xd8d8d0, 0xabab9c, 0x7c7c67, 0x5b5b4b, 0x474739, 0x2b2b22, 0x1d1d16, 0x0c0c09})
	return colors
}

func addPrimitiveScale(colors map[ColorToken]RGBAColor, tokens [11]ColorToken, values [11]uint32) {
	for index, token := range tokens {
		value := values[index]
		colors[token] = RGBA(uint8(value>>16), uint8(value>>8), uint8(value), 255)
	}
}
