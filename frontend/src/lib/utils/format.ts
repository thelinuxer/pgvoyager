// Format a Postgres temporal value (ISO-ish string) for display based on the
// column's data type. Returns null if the value doesn't match a known shape so
// the caller can fall back to its default rendering.
export function formatTemporal(value: string, dataType: string): string | null {
	const lower = dataType.toLowerCase();

	if (lower === 'date') {
		const m = value.match(/^(\d{4}-\d{2}-\d{2})/);
		return m ? m[1] : null;
	}

	if (lower.startsWith('timestamp')) {
		const m = value.match(
			/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2}:\d{2})(?:\.\d+)?(Z|[+-]\d{2}(?::?\d{2})?)?$/
		);
		if (!m) return null;
		const [, date, time, tz] = m;
		return `${date} ${time}` + (tz && tz !== 'Z' ? ' ' + tz : '');
	}

	if (lower.startsWith('time')) {
		const t = value.match(/^(\d{2}:\d{2}:\d{2})(?:\.\d+)?(Z|[+-]\d{2}(?::?\d{2})?)?$/);
		if (!t) return null;
		return t[1] + (t[2] && t[2] !== 'Z' ? ' ' + t[2] : '');
	}

	return null;
}

// Format a cell value for display in a data grid / preview. dataType is
// optional; when provided, temporal columns are normalized (e.g. `date`
// doesn't render the zero-time portion).
export function formatCellValue(value: unknown, dataType?: string): string {
	if (value === null) return 'NULL';
	if (value === undefined) return '';
	if (typeof value === 'object') return JSON.stringify(value);
	if (typeof value === 'string' && dataType) {
		const formatted = formatTemporal(value, dataType);
		if (formatted !== null) return formatted;
	}
	return String(value);
}
