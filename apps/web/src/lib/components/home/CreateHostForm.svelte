<script lang="ts">
	import { onMount } from 'svelte';
	import { InfoIcon } from 'lucide-svelte';
	import { toast } from 'svoast';
	import ServerError from '$lib/components/ui/ServerError.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Popover from '$lib/components/ui/popover/index';
	import { authStore } from '$lib/stores/auth';
	import { hostsStore } from '$lib/stores/hosts';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';
	import { isValidMACAddress, normalizeMACAddress } from '$lib/utils/mac';
	import { validateBroadcastAddress, validateIPv4 } from '$lib/utils/validation';
	import AdvancedSettings from './AdvancedSettings.svelte';
	import CreateHostFormSkeleton from './CreateHostFormSkeleton.svelte';

	let {
		class: className,
		onSuccess,
		isLoading = false
	}: { class?: string; onSuccess?: () => void; isLoading?: boolean } = $props();

	let systemConfig = $state<import('$lib/stores/hosts').AppConfig | null>(null);
	let selectedInterfaces = $state<string[]>([]);
	let isLoadingConfig = $state(false);
	let hasError = $state(false);
	let showSkeleton = $state(false);
	let skeletonTimeout: number | undefined;
	let minDisplayStartTime: number | null = null;
	let hasLoadedConfig = $state(false); // Track if config has been loaded

	let formData = $state({
		name: '',
		mac: '',
		broadcast: '255.255.255.255:9',
		interface: '',
		static_ip: '',
		use_as_fallback: false
	});

	let showAdvanced = $state(false);

	// Expose refresh function for parent component to call when dialog opens
	export async function refreshInterfaces() {
		// Don't reload if already loading or already loaded
		if (isLoadingConfig || hasLoadedConfig) {
			return;
		}

		// Only show loading if we actually need to fetch interfaces
		if (systemConfig && !systemConfig.supports_interface_selection) {
			return;
		}

		await loadConfig();
	}

	async function loadConfig() {
		// Prevent concurrent loads
		if (isLoadingConfig) {
			return;
		}

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

			// Load base config first
			systemConfig = await hostsStore.getConfig();
			// Then load network interfaces if needed (only if interface selection is supported)
			if (systemConfig.supports_interface_selection) {
				const interfaces = await hostsStore.getNetworkInterfaces();
				systemConfig.network_interfaces = interfaces;
			}
			hasError = false;
			hasLoadedConfig = true;
		} catch (err) {
			console.error('Failed to load system config:', err);
			hasError = true;
			hasLoadedConfig = false;
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

	onMount(async () => {
		await loadConfig();
	});

	// Update form data when selected interfaces change
	$effect(() => {
		const newValue = selectedInterfaces.join(',');
		if (formData.interface !== newValue) {
			formData.interface = newValue;
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();

		// Wait for config to load before checking auth
		if ($authStore.isLoading) {
			toast.error($t.messages.auth.loadingConfig, { closable: true });
			return;
		}

		// Only check authentication if auth is required
		if ($authStore.useAuth && !$authStore.isAuthenticated) {
			toast.error($t.messages.user.mustBeLoggedIn, { closable: true });
			return;
		}

		// Trim whitespace
		const trimmedData = {
			name: formData.name.trim(),
			mac: formData.mac.trim(),
			broadcast: formData.broadcast.trim(),
			interface: formData.interface.trim(),
			static_ip: formData.static_ip.trim(),
			use_as_fallback: formData.use_as_fallback
		};

		// Validate MAC address format
		if (!isValidMACAddress(trimmedData.mac)) {
			toast.error($t.messages.validation.invalidMac, { closable: true });
			return;
		}

		// Validate broadcast address format
		const broadcastValidation = validateBroadcastAddress(trimmedData.broadcast);
		if (!broadcastValidation.valid) {
			toast.error($t.messages.validation.invalidBroadcast, { closable: true });
			return;
		}

		// Validate static IP if provided
		if (trimmedData.static_ip && !validateIPv4(trimmedData.static_ip)) {
			toast.error($t.messages.validation.invalidIpv4, { closable: true });
			return;
		}

		// Normalize MAC address
		trimmedData.mac = normalizeMACAddress(trimmedData.mac);

		// Remove interface field if per-host interface selection is not supported
		const hostData: any = { ...trimmedData };
		if (!systemConfig?.supports_interface_selection) {
			delete hostData.interface;
		}

		// Remove static_ip and use_as_fallback if not provided
		if (!hostData.static_ip) {
			delete hostData.static_ip;
			delete hostData.use_as_fallback;
		}

		try {
			await hostsStore.createHost(hostData);
			toast.success($t.messages.host.createSuccess, { closable: true });
			// Reset form
			formData = {
				name: '',
				mac: '',
				broadcast: '255.255.255.255:9',
				interface: '',
				static_ip: '',
				use_as_fallback: false
			};
			selectedInterfaces = [];
			showAdvanced = false;
			onSuccess?.();
		} catch (error) {
			// Error is already handled in hostsStore.createHost
		}
	}
</script>

{#if hasError}
	<ServerError onRetry={loadConfig} class={cn('md:col-span-2', className)} />
{:else if isLoadingConfig || showSkeleton}
	<CreateHostFormSkeleton class={className} />
{:else}
	<form onsubmit={handleSubmit} class={cn('grid grid-cols-1 gap-4 md:grid-cols-2', className)}>
		<div class="flex flex-col gap-2">
			<label for="create-name" class="text-sm font-medium">{$t.ui.host.form.nameLabel}</label>
			<Input
				id="create-name"
				name="name"
				bind:value={formData.name}
				placeholder={$t.ui.host.form.namePlaceholder}
			/>
		</div>
		<div class="flex flex-col gap-2">
			<label for="create-mac" class="text-sm font-medium">{$t.ui.host.form.macLabel}</label>
			<div class="flex items-center space-x-1">
				<Input
					id="create-mac"
					name="mac"
					bind:value={formData.mac}
					placeholder={$t.ui.host.form.macPlaceholder}
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
			<label for="create-broadcast" class="text-sm font-medium"
				>{$t.ui.common.broadcastAddress}</label
			>
			<div class="flex items-center space-x-1">
				<Input
					id="create-broadcast"
					name="broadcast"
					bind:value={formData.broadcast}
					placeholder="255.255.255.255:9"
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
		<div class="md:col-span-2">
			<div class="border-t"></div>
		</div>

		<!-- Advanced Settings Toggle -->
		<div class="md:col-span-2">
			<button
				type="button"
				onclick={() => (showAdvanced = !showAdvanced)}
				class="flex w-full items-center justify-between rounded-md border px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
			>
				<span>{$t.ui.common.advancedSettings}</span>
				<svg
					class="h-4 w-4 transition-transform"
					class:rotate-180={showAdvanced}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"
					></path>
				</svg>
			</button>
		</div>

		<!-- Advanced Settings Content -->
		{#if showAdvanced}
			<AdvancedSettings
				{systemConfig}
				bind:selectedInterfaces
				bind:staticIp={formData.static_ip}
				bind:useAsFallback={formData.use_as_fallback}
				disabled={false}
				className="md:col-span-2"
			/>
		{/if}

		<Button type="submit" class={cn('my-1 w-full md:col-span-2')}
			>{$t.ui.host.form.createButton}</Button
		>
	</form>
{/if}
