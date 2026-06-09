package selfupdate

// Seams (overridable in tests); default to the platform implementations.
var (
	canElevateFn      = canElevate
	elevatedReplaceFn = elevatedReplace
)
