<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import {
		Activity,
		BellRing,
		Edit3,
		EllipsisVertical,
		InfoIcon,
		Trash2,
		Wifi,
		WifiOff
	} from 'lucide-svelte';
	import { toast } from 'svoast';
	import ServerError from '$lib/components/ui/ServerError.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Input } from '$lib/components/ui/input';
	import { MultiSelect } from '$lib/components/ui/multi-select';
	import * as Popover from '$lib/components/ui/popover/index';
	import { authStore } from '$lib/stores/auth';
	import type { AppConfig, Host } from '$lib/stores/hosts';
	import { hostsStore } from '$lib/stores/hosts';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';
	import { isValidMACAddress, normalizeMACAddress } from '$lib/utils/mac';
	import { validateBroadcastAddress, validateHostName, validateIPv4 } from '$lib/utils/validation';
	import AdvancedSettings from './AdvancedSettings.svelte';
	import HostCardEditSkeleton from './HostCardEditSkeleton.svelte';

	type Props = {
		host: Host;
		class?: string;
		bulkPingResult?:
			| { ping_success: boolean; arp_success: boolean; server_unreachable?: boolean }
			| undefined;
	};

	let { host, class: className = '', bulkPingResult = undefined }: Props = $props();

	type HostStatus = {
		ping_success: boolean;
		arp_success: boolean;
		rate_limited?: boolean;
		server_unreachable?: boolean;
		timestamp: number;
		loading: boolean;
	};

	let status: HostStatus = $state({
		ping_success: false,
		arp_success: false,
		timestamp: 0,
		loading: false
	});

	// Track the loading start time to enforce minimum display time
	let loadingStartTime: number | null = null;
	// Store bulk ping result that arrived during manual ping operation
	let pendingBulkResult: import('$lib/stores/hosts').PingResult | null = null;

	let systemConfig: AppConfig | null = $state(null);
	let isEditing = $state(false);
	let showDeleteModal = $state(false);
	let showOptionsMenu = $state(false);
	let isLoadingConfig = $state(false);
	let hasError = $state(false);
	let showSkeleton = $state(false);
	let skeletonTimeout: number | undefined;
	let minDisplayStartTime: number | null = null;
	let showAdvanced = $state(false);
	let editData = $state({
		name: '',
		mac: '',
		broadcast: '255.255.255.255:9',
		interface: '',
		static_ip: '',
		use_as_fallback: false
	});
	let selectedInterfaces = $state<string[]>([]);

	// Update editData.interface when selectedInterfaces changes
	$effect(() => {
		if (isEditing) {
			const newValue = selectedInterfaces.join(',');
			if (editData.interface !== newValue) {
				editData.interface = newValue;
			}
		}
	});

	let menuButtonRef: HTMLButtonElement | null = $state(null);

	function wake() {
		hostsStore.wakeHost(host);
	}

	// Close menu when clicking outside
	$effect(() => {
		if (!showOptionsMenu) return;

		function handleClickOutside(event: MouseEvent) {
			const target = event.target as Node;
			// Don't close if clicking the menu button or inside the menu
			if (menuButtonRef?.contains(target)) return;
			showOptionsMenu = false;
		}

		document.addEventListener('click', handleClickOutside);
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	function showDeleteConfirmation() {
		showDeleteModal = true;
	}

	function confirmDelete() {
		hostsStore.deleteHost(host);
		showDeleteModal = false;
	}

	function closeModal() {
		showDeleteModal = false;
	}

	async function pingHost() {
		status.loading = true;
		loadingStartTime = Date.now();

		try {
			const result = await hostsStore.pingHost(host);

			// Calculate elapsed time and ensure minimum 250ms display for loading state
			const elapsed = Date.now() - (loadingStartTime || Date.now());
			const remainingTime = Math.max(0, 250 - elapsed);

			// Wait for the remaining time if needed to ensure minimum display
			if (remainingTime > 0) {
				await new Promise((resolve) => setTimeout(resolve, remainingTime));
			}

			// Apply pending bulk ping result if available, otherwise use the manual ping result
			const finalResult = pendingBulkResult || result;

			status = {
				ping_success: finalResult.ping_success,
				arp_success: finalResult.arp_success,
				rate_limited: finalResult.rate_limited,
				server_unreachable: finalResult.server_unreachable,
				timestamp: Date.now(),
				loading: false
			};

			// Clear the pending bulk result
			pendingBulkResult = null;
			loadingStartTime = null;
		} catch (err) {
			// Calculate elapsed time and ensure minimum 250ms display for loading state
			const elapsed = Date.now() - (loadingStartTime || Date.now());
			const remainingTime = Math.max(0, 250 - elapsed);

			// Wait for the remaining time if needed to ensure minimum display
			if (remainingTime > 0) {
				await new Promise((resolve) => setTimeout(resolve, remainingTime));
			}

			status.loading = false;
			// Clear the pending bulk result and loading start time
			pendingBulkResult = null;
			loadingStartTime = null;
			console.error('Failed to ping host:', err);
		}
	}

	// Don't load config on mount - only load when needed (when editing)
	// This prevents unnecessary API calls on page load

	// Update status when bulk ping result changes
	$effect(() => {
		if (bulkPingResult && bulkPingResult.ping_success !== undefined) {
			// If we're currently in a manual ping operation (with minimum display time),
			// store the bulk result but don't update the status yet
			if (loadingStartTime) {
				// Store the bulk result for later use after minimum loading time
				pendingBulkResult = bulkPingResult;
			} else {
				// Update the status with the bulk ping result
				status = {
					ping_success: bulkPingResult.ping_success,
					arp_success: bulkPingResult.arp_success,
					rate_limited: false,
					server_unreachable: bulkPingResult.server_unreachable,
					timestamp: Date.now(),
					loading: false
				};
				// Clear loading start time if we were in a loading state
				loadingStartTime = null;
			}
		}
	});

	function getStatusColor() {
		if (status.loading) return 'text-yellow-500';
		if (status.rate_limited) return 'text-orange-500';
		if (status.server_unreachable) return 'text-gray-500';
		return status.ping_success ? 'text-green-500' : 'text-red-500';
	}

	function getStatusIcon() {
		if (status.loading) return Activity;
		if (status.server_unreachable) return WifiOff;
		return status.ping_success ? Wifi : WifiOff;
	}

	function getStatusText() {
		if (status.loading) return $t.ui.host.card.checking;
		if (status.rate_limited) return $t.ui.host.card.rateLimited;
		if (status.server_unreachable) return $t.ui.host.card.serverUnreachable;
		return status.ping_success ? $t.ui.host.card.online : $t.ui.host.card.offline;
	}

	function formatTimestamp(timestamp: number) {
		if (!timestamp) return $t.ui.host.card.never;
		return new Date(timestamp * 1000).toLocaleTimeString();
	}

	async function startEdit() {
		isEditing = true;
		showSkeleton = true; // Show skeleton immediately when edit starts
		isLoadingConfig = true;
		systemConfig = null; // Clear cached config to force fresh fetch
		hasError = false; // Reset error state

		editData = {
			name: host.name,
			mac: host.mac,
			broadcast: host.broadcast || '255.255.255.255:9',
			interface: host.interface || '',
			static_ip: host.static_ip || '',
			use_as_fallback: host.use_as_fallback || false
		};

		// Parse selected interfaces from host.interface
		selectedInterfaces = host.interface
			? host.interface
					.split(',')
					.map((i: string) => i.trim())
					.filter(Boolean)
			: [];

		// Set showAdvanced if static IP is configured
		showAdvanced = !!host.static_ip;

		// Load config with error handling
		await loadEditConfig();
	}

	async function loadEditConfig() {
		isLoadingConfig = true;
		hasError = false;
		showSkeleton = true;

		// Track when skeleton display started for minimum display time
		minDisplayStartTime = Date.now();

		try {
			// Check if server is reachable first
			if ($authStore.serverUnreachable) {
				throw new Error('Server unreachable');
			}

			// Load config if not already loaded
			if (!systemConfig) {
				systemConfig = await hostsStore.getConfig();
			}
			// Only fetch network interfaces if interface selection is supported
			if (systemConfig?.supports_interface_selection) {
				const interfaces = await hostsStore.getNetworkInterfaces();
				if (systemConfig) {
					systemConfig.network_interfaces = interfaces;
					systemConfig.supports_interface_selection = interfaces.length > 0;
				}
			}
			hasError = false;
		} catch (err) {
			console.error('Failed to refresh network interfaces:', err);
			hasError = true;
			systemConfig = null; // Clear cached config on error to force fresh fetch on retry
		} finally {
			// Ensure skeleton is shown for at least 150ms to prevent blink effect
			const elapsedTime = Date.now() - (minDisplayStartTime || 0);
			const remainingTime = Math.max(0, 150 - elapsedTime);

			if (remainingTime > 0) {
				await new Promise((resolve) => setTimeout(resolve, remainingTime));
			}

			clearTimeout(skeletonTimeout);
			isLoadingConfig = false;
			showSkeleton = false;
			minDisplayStartTime = null;
		}
	}

	function cancelEdit() {
		isEditing = false;
		showAdvanced = false;
		hasError = false;
		showSkeleton = false;
		systemConfig = null; // Clear cached config to force fresh fetch on next edit
		clearTimeout(skeletonTimeout);
	}

	async function saveEdit() {
		// Trim name
		const trimmedName = editData.name.trim();

		// Validate Host Name
		const nameValidation = validateHostName(trimmedName);
		if (!nameValidation.valid) {
			const errorCode = nameValidation.error as keyof typeof $t.messages.error.codes;
			toast.error($t.messages.error.codes[errorCode] || $t.messages.error.generic, { closable: true });
			return;
		}

		// Validate MAC address
		if (!isValidMACAddress(editData.mac)) {
			toast.error($t.messages.error.codes.ERR_INVALID_MAC, { closable: true });
			return;
		}

		// Validate broadcast address format
		const broadcastValidation = validateBroadcastAddress(editData.broadcast);
		if (!broadcastValidation.valid) {
			toast.error($t.messages.error.codes.ERR_INVALID_BROADCAST, { closable: true });
			return;
		}

		// Validate static IP if provided
		if (editData.static_ip && !validateIPv4(editData.static_ip)) {
			toast.error($t.messages.error.codes.ERR_INVALID_IP, { closable: true });
			return;
		}

		try {
			const updatedHost: Host = {
				...host,
				name: trimmedName,
				mac: normalizeMACAddress(editData.mac),
				broadcast: editData.broadcast,
				static_ip: editData.static_ip || undefined,
				use_as_fallback: editData.static_ip ? editData.use_as_fallback : false
			};

			// Only include interface if per-host interface selection is supported
			if (systemConfig?.supports_interface_selection) {
				updatedHost.interface = editData.interface || undefined;
			}

			await hostsStore.updateHost(updatedHost);
			toast.success($t.messages.host.updateSuccess, { closable: true });
			isEditing = false;
			showAdvanced = false;

			// Ping the host immediately after successful update
			pingHost();
		} catch (error: any) {
			// Error is handled in hostsStore.updateHost
		}
	}
</script>

{#if host}
	<!-- <Card.Root class={cn('relative bg-accent/80 dark:bg-accent/20', className)}> -->
	<Card.Root class={cn('', className)}>
		{#if isEditing}
			{#if hasError}
				<Card.Content>
					<ServerError onRetry={loadEditConfig} />
				</Card.Content>
				<Card.Footer class="flex justify-end gap-2">
					<Button type="button" variant="outline" onclick={cancelEdit}>{$t.ui.common.cancel}</Button
					>
				</Card.Footer>
			{:else if showSkeleton}
				<HostCardEditSkeleton />
			{:else}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						saveEdit();
					}}
				>
					<Card.Content>
						<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
							<div class="flex flex-col gap-2">
								<label for="edit-name-{host.id}" class="text-sm font-medium"
									>{$t.ui.host.form.editNameLabel}</label
								>
								<Input
									id="edit-name-{host.id}"
									bind:value={editData.name}
									placeholder={$t.ui.host.form.deviceNamePlaceholder}
								/>
							</div>
							<div class="flex flex-col gap-2">
								<label for="edit-mac-{host.id}" class="text-sm font-medium"
									>{$t.ui.host.form.editMacLabel}</label
								>
								<div class="flex items-center space-x-1">
									<Input
										id="edit-mac-{host.id}"
										bind:value={editData.mac}
										placeholder={$t.ui.host.form.macAddressPlaceholder}
										class="flex-1"
									/>
									<Popover.Root>
										<Popover.Trigger>
											<Button variant="secondary" size="icon" type="button" class="shrink-0"
												><InfoIcon class="h-4 w-4" /></Button
											>
										</Popover.Trigger>
										<Popover.Content>
											<p class="mb-2 text-sm font-semibold">{$t.ui.host.info.macTitle}</p>
											<p class="mb-1 text-sm">{$t.ui.host.info.macDescription}</p>
											<ul class="ml-4 list-disc space-y-1 text-sm">
												<li><code>AA:BB:CC:DD:EE:FF</code> ({$t.ui.host.info.macFormat1})</li>
												<li><code>AA-BB-CC-DD-EE-FF</code> ({$t.ui.host.info.macFormat2})</li>
											</ul>
											<p class="mt-2 text-sm text-muted-foreground">
												{$t.ui.host.info.macNote}
											</p>
										</Popover.Content>
									</Popover.Root>
								</div>
							</div>
							<div class="flex flex-col gap-2 md:col-span-2">
								<label for="edit-broadcast-{host.id}" class="text-sm font-medium"
									>{$t.ui.host.form.editBroadcastLabel}</label
								>
								<div class="flex items-center space-x-1">
									<Input
										id="edit-broadcast-{host.id}"
										bind:value={editData.broadcast}
										placeholder={$t.ui.host.form.editBroadcastPlaceholder}
										class="flex-1"
									/>
									<Popover.Root>
										<Popover.Trigger>
											<Button variant="secondary" size="icon" type="button" class="shrink-0"
												><InfoIcon class="h-4 w-4" /></Button
											>
										</Popover.Trigger>
										<Popover.Content>
											<p class="mb-2 text-sm font-semibold">{$t.ui.host.info.broadcastTitle}</p>
											<p class="mb-1 text-sm">
												{$t.ui.host.info.broadcastFormat}
											</p>
											<p class="mb-1 text-sm">
												{$t.ui.host.info.broadcastDefault}
											</p>
											<p class="mb-1 text-sm">
												{$t.ui.host.info.broadcastMultiNetwork}
											</p>
											<p class="text-sm text-muted-foreground">
												{$t.ui.host.info.broadcastExample}
											</p>
										</Popover.Content>
									</Popover.Root>
								</div>
							</div>

							<!-- Divider -->
							<div class="col-span-2">
								<div class="border-t"></div>
							</div>

							<!-- Advanced Settings Toggle -->
							<div class="col-span-2">
								<button
									type="button"
									onclick={() => (showAdvanced = !showAdvanced)}
									class="flex w-full items-center justify-between rounded-md border px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
								>
									<span>{$t.ui.host.form.editAdvancedSettings}</span>
									<svg
										class="h-4 w-4 transition-transform"
										class:rotate-180={showAdvanced}
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M19 9l-7 7-7-7"
										></path>
									</svg>
								</button>
							</div>

							<!-- Advanced Settings Content -->
							{#if showAdvanced}
								{#if isLoadingConfig}
									<div class="col-span-2 flex flex-col gap-2">
										<div class="h-5 w-48 animate-pulse rounded bg-muted"></div>
										<div class="h-10 w-full animate-pulse rounded-md bg-muted"></div>
										<div class="h-3 w-full animate-pulse rounded bg-muted"></div>
									</div>
								{:else}
									<AdvancedSettings
										{systemConfig}
										bind:selectedInterfaces
										bind:staticIp={editData.static_ip}
										bind:useAsFallback={editData.use_as_fallback}
										disabled={false}
										className="col-span-2"
									/>
								{/if}
							{/if}
						</div>
					</Card.Content>
					<Card.Footer class="flex justify-end gap-2">
						<Button type="button" variant="outline" onclick={cancelEdit}
							>{$t.ui.host.form.cancelButton}</Button
						>
						<Button type="submit">{$t.ui.host.form.saveButton}</Button>
					</Card.Footer>
				</form>
			{/if}
		{:else}
			<Card.Content>
				<div class={cn('grid grid-cols-2 gap-2', className)}>
					<p class="text-md flex min-w-0 items-baseline gap-1 font-bold">
						<span class="shrink-0">{$t.ui.host.card.nameLabel}:</span><span
							class="truncate font-mono font-medium"
							title={host.name}>{host.name}</span
						>
					</p>
					<div
						class="flex items-center justify-end"
						title="Last ping: {formatTimestamp(status.timestamp / 1000)}"
					>
						<span class="relative flex h-3 w-3">
							{#if status.loading}
								<span
									class="absolute inline-flex h-full w-full animate-ping rounded-full bg-yellow-400 opacity-75"
								></span>
								<span class="relative inline-flex h-3 w-3 rounded-full bg-yellow-500"></span>
							{:else if status.server_unreachable}
								<span class="relative inline-flex h-3 w-3 rounded-full bg-gray-500"></span>
							{:else if status.ping_success}
								<span
									class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"
								></span>
								<span class="relative inline-flex h-3 w-3 rounded-full bg-green-500"></span>
							{:else}
								<span class="relative inline-flex h-3 w-3 rounded-full bg-red-500"></span>
							{/if}
						</span>
					</div>
					{#if $authStore.showSensitiveData}
						<p class="text-md col-span-2 font-bold">
							{$t.ui.host.card.broadcastLabel}
							<span class="font-mono font-medium">{host.broadcast}</span>
						</p>
						<p class="text-md col-span-2 font-bold">
							{$t.ui.host.card.macLabel}
							<span class="font-mono font-medium">{host.mac.toLowerCase()}</span>
						</p>
						{#if host.interface}
							<p class="text-md col-span-2 font-bold">
								{$t.ui.host.card.interfacesLabel}
								<span class="font-mono font-medium text-muted-foreground"
									>{host.interface.replace(/,/g, ', ')}</span
								>
							</p>
						{:else}
							<p class="text-md col-span-2 font-bold">
								{$t.ui.host.card.interfacesLabel}
								<span class="font-mono font-medium text-muted-foreground"
									>{$t.ui.host.card.allInterfaces}</span
								>
							</p>
						{/if}
					{/if}
				</div>
			</Card.Content>
			<Card.Footer class="relative flex justify-between gap-2">
				<Button
					size="sm"
					variant="outline"
					class="flex-1 bg-green-500/20 hover:bg-green-500/30"
					onclick={wake}
				>
					<BellRing class="mr-2 h-4 w-4" />
					{$t.ui.host.card.wake}
				</Button>
				<Button
					size="sm"
					variant="outline"
					class="flex-1"
					onclick={pingHost}
					disabled={status.loading}
				>
					{@const StatusIcon = getStatusIcon()}
					<StatusIcon class={cn('mr-2 h-4 w-4', getStatusColor())} />
					{status.loading ? $t.ui.host.card.pingingButton : $t.ui.host.card.pingButton}
				</Button>
				{#if !$authStore.isReadOnly}
					<Button
						bind:ref={menuButtonRef}
						size="sm"
						variant="outline"
						onclick={() => (showOptionsMenu = !showOptionsMenu)}
						title="More options"
					>
						<EllipsisVertical class="h-4 w-4" />
					</Button>
					{#if showOptionsMenu}
						<div
							class="absolute bottom-full right-0 z-10 mb-1 min-w-[120px] rounded-md border bg-popover shadow-lg"
						>
							<button
								class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground"
								onclick={() => {
									startEdit();
									showOptionsMenu = false;
								}}
							>
								<Edit3 class="h-4 w-4" />
								{$t.ui.host.card.edit}
							</button>
							<button
								class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-accent dark:text-red-500"
								onclick={() => {
									showDeleteConfirmation();
									showOptionsMenu = false;
								}}
							>
								<Trash2 class="h-4 w-4" />
								{$t.ui.host.card.delete}
							</button>
						</div>
					{/if}
				{/if}
			</Card.Footer>
		{/if}
	</Card.Root>

	{#if showDeleteModal}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			transition:fade={{ duration: 150 }}
			class="modal-backdrop"
			onclick={(e) => e.target === e.currentTarget && closeModal()}
			onkeydown={(e) => e.key === 'Escape' && closeModal()}
			role="dialog"
			aria-modal="true"
			tabindex="-1"
		>
			<Card.Root class="mx-4 w-full max-w-md border-2 shadow-2xl">
				<Card.Header class="pt-6">
					<Card.Title>{$t.ui.host.card.deleteTitle}</Card.Title>
					<Card.Description>{$t.ui.host.card.deleteDescription}</Card.Description>
				</Card.Header>
				<Card.Footer class="flex justify-end gap-2">
					<Button variant="outline" onclick={closeModal}>{$t.ui.common.cancel}</Button>
					<Button variant="destructive" onclick={confirmDelete}>{$t.ui.host.card.delete}</Button>
				</Card.Footer>
			</Card.Root>
		</div>
	{/if}

	<style>
		.modal-backdrop {
			position: fixed;
			inset: 0;
			z-index: 9999;
			display: flex;
			align-items: center;
			justify-content: center;
			background: rgba(0, 0, 0, 0.2);
			backdrop-filter: blur(4px);
		}
	</style>
{/if}
