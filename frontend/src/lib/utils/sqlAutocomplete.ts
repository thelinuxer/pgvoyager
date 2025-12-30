import type { CompletionContext, CompletionResult, Completion } from '@codemirror/autocomplete';
import type { Table, View, Function, Column } from '$lib/types';

// SQL keywords for PostgreSQL
const SQL_KEYWORDS = [
	'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'LIKE', 'ILIKE', 'BETWEEN', 'IS', 'NULL',
	'JOIN', 'INNER', 'LEFT', 'RIGHT', 'FULL', 'OUTER', 'CROSS', 'ON', 'USING',
	'ORDER', 'BY', 'ASC', 'DESC', 'NULLS', 'FIRST', 'LAST',
	'GROUP', 'HAVING', 'LIMIT', 'OFFSET', 'DISTINCT', 'ALL',
	'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE',
	'CREATE', 'ALTER', 'DROP', 'TABLE', 'VIEW', 'INDEX',
	'AS', 'UNION', 'INTERSECT', 'EXCEPT',
	'CASE', 'WHEN', 'THEN', 'ELSE', 'END', 'CAST', 'COALESCE',
	'EXISTS', 'TRUE', 'FALSE', 'DEFAULT', 'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES',
	'WITH', 'RETURNING', 'TRUNCATE', 'CASCADE', 'RESTRICT',
	'NATURAL', 'LATERAL', 'ONLY', 'OVER', 'PARTITION', 'WINDOW',
	'FETCH', 'NEXT', 'ROWS', 'PERCENT', 'TIES',
	'UNIQUE', 'CHECK', 'CONSTRAINT', 'NOT NULL', 'SERIAL', 'GENERATED', 'ALWAYS', 'IDENTITY'
];

// PostgreSQL data types
const PG_TYPES = [
	'integer', 'int', 'int4', 'bigint', 'int8', 'smallint', 'int2',
	'serial', 'bigserial', 'smallserial',
	'text', 'varchar', 'character varying', 'char', 'character',
	'boolean', 'bool',
	'date', 'time', 'timestamp', 'timestamptz', 'timestamp with time zone', 'interval',
	'numeric', 'decimal', 'real', 'float4', 'double precision', 'float8',
	'json', 'jsonb', 'uuid', 'bytea', 'xml',
	'inet', 'cidr', 'macaddr', 'money',
	'point', 'line', 'lseg', 'box', 'path', 'polygon', 'circle',
	'tsquery', 'tsvector', 'oid'
];

export interface SchemaData {
	tables: Table[];
	views: View[];
	functions: Function[];
	columns: ColumnInfo[];
}

export interface ColumnInfo {
	name: string;
	tableName: string;
	schema: string;
	dataType: string;
}

function buildCompletions(schemaData: SchemaData): Completion[] {
	const completions: Completion[] = [];

	// Add tables
	for (const table of schemaData.tables) {
		const label = table.schema === 'public' ? table.name : `${table.schema}.${table.name}`;
		completions.push({
			label,
			type: 'class',
			detail: 'table',
			boost: 2
		});
	}

	// Add views
	for (const view of schemaData.views) {
		const label = view.schema === 'public' ? view.name : `${view.schema}.${view.name}`;
		completions.push({
			label,
			type: 'interface',
			detail: 'view',
			boost: 2
		});
	}

	// Add functions
	for (const func of schemaData.functions) {
		const label = func.schema === 'public' ? func.name : `${func.schema}.${func.name}`;
		completions.push({
			label: `${label}()`,
			type: 'function',
			detail: func.returnType || 'function',
			boost: 1
		});
	}

	// Add columns
	for (const col of schemaData.columns) {
		completions.push({
			label: col.name,
			type: 'property',
			detail: `${col.tableName}.${col.name} (${col.dataType})`,
			boost: 3
		});
	}

	// Add SQL keywords
	for (const keyword of SQL_KEYWORDS) {
		completions.push({
			label: keyword,
			type: 'keyword',
			boost: 0
		});
		// Also add lowercase version
		completions.push({
			label: keyword.toLowerCase(),
			type: 'keyword',
			boost: 0
		});
	}

	// Add PostgreSQL types
	for (const pgType of PG_TYPES) {
		completions.push({
			label: pgType,
			type: 'type',
			boost: 0
		});
	}

	return completions;
}

export function createSchemaCompletionSource(schemaData: SchemaData) {
	const completions = buildCompletions(schemaData);

	return function schemaCompletionSource(context: CompletionContext): CompletionResult | null {
		// Get the word before cursor
		const word = context.matchBefore(/[\w."]+/);

		// Don't show completions if no word and not explicitly triggered
		if (!word && !context.explicit) {
			return null;
		}

		const from = word ? word.from : context.pos;
		const text = word ? word.text.toLowerCase() : '';

		// Filter completions based on input
		const filtered = completions.filter(c =>
			c.label.toLowerCase().startsWith(text) ||
			c.label.toLowerCase().includes(text)
		);

		// Sort: exact prefix matches first, then contains matches
		filtered.sort((a, b) => {
			const aStartsWith = a.label.toLowerCase().startsWith(text);
			const bStartsWith = b.label.toLowerCase().startsWith(text);
			if (aStartsWith && !bStartsWith) return -1;
			if (!aStartsWith && bStartsWith) return 1;
			// Then sort by boost (higher first)
			return (b.boost || 0) - (a.boost || 0);
		});

		return {
			from,
			options: filtered,
			validFor: /^[\w."]*$/
		};
	};
}
