<script lang="ts">
	import { activeConnectionId } from '$lib/stores/connections';
	import { analysisApi } from '$lib/api/client';
	import type { Tab, AnalysisResult, AnalysisCategory, AnalysisIssue } from '$lib/types';
	import Icon from '$lib/icons/Icon.svelte';

	interface Props {
		tab: Tab;
	}

	let { tab }: Props = $props();

	let analysisResult = $state<AnalysisResult | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let expandedCategories = $state<Set<string>>(new Set());
	let copiedSuggestion = $state<string | null>(null);

	$effect(() => {
		if ($activeConnectionId) {
			runAnalysis();
		}
	});

	async function runAnalysis() {
		if (!$activeConnectionId) return;

		isLoading = true;
		error = null;

		try {
			analysisResult = await analysisApi.run($activeConnectionId);
			// Expand categories with issues by default
			expandedCategories = new Set(analysisResult.categories.map((c) => c.name));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to run analysis';
		} finally {
			isLoading = false;
		}
	}

	function toggleCategory(name: string) {
		if (expandedCategories.has(name)) {
			expandedCategories.delete(name);
		} else {
			expandedCategories.add(name);
		}
		expandedCategories = new Set(expandedCategories);
	}

	function getSeverityIcon(severity: string): string {
		switch (severity) {
			case 'critical':
				return 'alert-circle';
			case 'warning':
				return 'alert-triangle';
			case 'info':
				return 'info';
			default:
				return 'info';
		}
	}

	function getCategoryIcon(icon: string): string {
		switch (icon) {
			case 'zap':
				return 'zap';
			case 'table':
				return 'table';
			case 'link':
				return 'link';
			case 'hash':
				return 'hash';
			case 'activity':
				return 'activity';
			default:
				return 'folder';
		}
	}

	async function copySuggestion(suggestion: string) {
		try {
			await navigator.clipboard.writeText(suggestion);
			copiedSuggestion = suggestion;
			setTimeout(() => {
				copiedSuggestion = null;
			}, 2000);
		} catch {
			// Clipboard API not available
		}
	}
</script>

<div class="analysis-viewer">
	<div class="toolbar">
		<div class="toolbar-left">
			<div class="breadcrumb">
				<Icon name="activity" size={14} />
				<span class="title">Database Analysis</span>
			</div>
		</div>
		<div class="toolbar-center">
			{#if analysisResult}
				<div class="summary">
					{#if analysisResult.summary.critical > 0}
						<span class="summary-item critical">
							<Icon name="alert-circle" size={12} />
							{analysisResult.summary.critical}
						</span>
					{/if}
					{#if analysisResult.summary.warning > 0}
						<span class="summary-item warning">
							<Icon name="alert-triangle" size={12} />
							{analysisResult.summary.warning}
						</span>
					{/if}
					{#if analysisResult.summary.info > 0}
						<span class="summary-item info">
							<Icon name="info" size={12} />
							{analysisResult.summary.info}
						</span>
					{/if}
					{#if analysisResult.summary.critical === 0 && analysisResult.summary.warning === 0 && analysisResult.summary.info === 0}
						<span class="summary-item ok">
							<Icon name="check-circle" size={12} />
							All checks passed
						</span>
					{/if}
				</div>
			{/if}
		</div>
		<div class="toolbar-right">
			<button class="btn btn-sm btn-ghost" onclick={runAnalysis} disabled={isLoading}>
				<Icon name="refresh" size={14} class={isLoading ? 'spinning' : ''} />
				Refresh
			</button>
		</div>
	</div>

	{#if isLoading && !analysisResult}
		<div class="loading">
			<Icon name="loader" size={24} class="spinning" />
			<span>Analyzing database...</span>
		</div>
	{:else if error}
		<div class="error">
			<Icon name="alert-circle" size={20} />
			{error}
		</div>
	{:else if analysisResult}
		<div class="content">
			<!-- Database Stats -->
			<div class="stats-section">
				<div class="stat-card">
					<span class="stat-value">{analysisResult.stats.databaseSize}</span>
					<span class="stat-label">Database Size</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">{analysisResult.stats.tableCount}</span>
					<span class="stat-label">Tables</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">{analysisResult.stats.indexCount}</span>
					<span class="stat-label">Indexes</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">{analysisResult.stats.cacheHitRatio}%</span>
					<span class="stat-label">Cache Hit Ratio</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">{analysisResult.stats.activeConnections}</span>
					<span class="stat-label">Active Connections</span>
				</div>
			</div>

			<!-- Issues by Category -->
			{#if analysisResult.categories.length === 0}
				<div class="no-issues">
					<Icon name="check-circle" size={48} />
					<h3>No Issues Found</h3>
					<p>Your database looks healthy!</p>
				</div>
			{:else}
				<div class="categories">
					{#each analysisResult.categories as category}
						<div class="category">
							<button
								class="category-header"
								onclick={() => toggleCategory(category.name)}
							>
								<div class="category-title">
									<Icon name={getCategoryIcon(category.icon)} size={16} />
									<span>{category.name}</span>
									<span class="issue-count">{category.issues.length}</span>
								</div>
								<Icon
									name="chevron-down"
									size={16}
									class={expandedCategories.has(category.name) ? 'expanded' : ''}
								/>
							</button>

							{#if expandedCategories.has(category.name)}
								<div class="issues">
									{#each category.issues as issue}
										<div class="issue severity-{issue.severity}">
											<div class="issue-header">
												<Icon name={getSeverityIcon(issue.severity)} size={16} />
												<span class="issue-title">{issue.title}</span>
											</div>
											<div class="issue-body">
												<p class="issue-description">{issue.description}</p>
												{#if issue.table}
													<div class="issue-meta">
														<span class="meta-label">Table:</span>
														<code>{issue.table}</code>
													</div>
												{/if}
												{#if issue.impact}
													<div class="issue-meta">
														<span class="meta-label">Impact:</span>
														<span>{issue.impact}</span>
													</div>
												{/if}
												{#if issue.suggestion}
													<div class="suggestion">
														<code>{issue.suggestion}</code>
														<button
															class="copy-btn"
															onclick={() => copySuggestion(issue.suggestion!)}
															title="Copy SQL"
														>
															<Icon
																name={copiedSuggestion === issue.suggestion ? 'check' : 'copy'}
																size={12}
															/>
														</button>
													</div>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{:else}
		<div class="not-connected">
			<Icon name="database" size={48} />
			<p>Connect to a database to run analysis</p>
		</div>
	{/if}
</div>

<style>
	.analysis-viewer {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 16px;
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
	}

	.toolbar-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.toolbar-center {
		display: flex;
		align-items: center;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.breadcrumb :global(svg) {
		color: var(--color-text-muted);
	}

	.title {
		font-weight: 600;
	}

	.summary {
		display: flex;
		gap: 12px;
	}

	.summary-item {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		padding: 4px 8px;
		border-radius: var(--radius-sm);
	}

	.summary-item.critical {
		background: rgba(239, 68, 68, 0.15);
		color: var(--color-error);
	}

	.summary-item.warning {
		background: rgba(245, 158, 11, 0.15);
		color: var(--color-warning);
	}

	.summary-item.info {
		background: rgba(59, 130, 246, 0.15);
		color: var(--color-info);
	}

	.summary-item.ok {
		background: rgba(34, 197, 94, 0.15);
		color: var(--color-success);
	}

	.spinning {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.loading,
	.error,
	.not-connected {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}

	.content {
		flex: 1;
		overflow: auto;
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.stats-section {
		display: flex;
		gap: 12px;
		flex-wrap: wrap;
	}

	.stat-card {
		flex: 1;
		min-width: 120px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: 16px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
	}

	.stat-value {
		font-size: 24px;
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.stat-label {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.no-issues {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		color: var(--color-success);
	}

	.no-issues h3 {
		margin: 0;
		font-size: 18px;
	}

	.no-issues p {
		margin: 0;
		color: var(--color-text-muted);
	}

	.categories {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.category {
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		overflow: hidden;
	}

	.category-header {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		background: transparent;
		border: none;
		cursor: pointer;
		color: var(--color-text);
	}

	.category-header:hover {
		background: var(--color-bg-tertiary);
	}

	.category-title {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 500;
	}

	.issue-count {
		font-size: 12px;
		padding: 2px 8px;
		background: var(--color-bg-tertiary);
		border-radius: 10px;
		color: var(--color-text-muted);
	}

	.category-header :global(.expanded) {
		transform: rotate(180deg);
	}

	.issues {
		border-top: 1px solid var(--color-border);
	}

	.issue {
		padding: 12px 16px;
		border-bottom: 1px solid var(--color-border);
	}

	.issue:last-child {
		border-bottom: none;
	}

	.issue-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 8px;
	}

	.issue.severity-critical .issue-header :global(svg) {
		color: var(--color-error);
	}

	.issue.severity-warning .issue-header :global(svg) {
		color: var(--color-warning);
	}

	.issue.severity-info .issue-header :global(svg) {
		color: var(--color-info);
	}

	.issue-title {
		font-weight: 500;
	}

	.issue-body {
		padding-left: 24px;
	}

	.issue-description {
		margin: 0 0 8px 0;
		color: var(--color-text-muted);
		font-size: 13px;
	}

	.issue-meta {
		display: flex;
		gap: 8px;
		font-size: 12px;
		margin-bottom: 4px;
	}

	.meta-label {
		color: var(--color-text-muted);
	}

	.issue-meta code {
		font-family: var(--font-mono);
		background: var(--color-bg-tertiary);
		padding: 1px 4px;
		border-radius: 3px;
	}

	.suggestion {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 8px;
		background: var(--color-bg-tertiary);
		padding: 8px 12px;
		border-radius: var(--radius-sm);
	}

	.suggestion code {
		flex: 1;
		font-family: var(--font-mono);
		font-size: 12px;
		color: var(--color-primary);
	}

	.copy-btn {
		padding: 4px;
		background: transparent;
		border: none;
		cursor: pointer;
		color: var(--color-text-muted);
		border-radius: var(--radius-sm);
	}

	.copy-btn:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}
</style>
