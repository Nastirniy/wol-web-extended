<script lang="ts">
	import { InfoIcon } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { MultiSelect } from '$lib/components/ui/multi-select';
	import * as Popover from '$lib/components/ui/popover/index';
	import type { AppConfig } from '$lib/stores/hosts';
	import { t } from '$lib/stores/locale';
	import { cn } from '$lib/utils';

	let {
		systemConfig,
		selectedInterfaces = $bindable(),
		staticIp = $bindable(),
		useAsFallback = $bindable(),
		disabled = false,
		className = ''
	}: {
		systemConfig: AppConfig | null;
		selectedInterfaces?: string[];
		staticIp?: string;
		useAsFallback?: boolean;
		disabled?: boolean;
		className?: string;
	} = $props();
</script>

<div class={cn('flex flex-col gap-4', className)}>
	<!-- Network Interfaces Section -->
	<div class="flex w-full flex-col gap-2">
		<label for="interface" class="text-sm font-medium">
			{$t.ui.common.networkInterfacesOptional}
			{#if !systemConfig?.supports_interface_selection}
				<span class="text-xs text-muted-foreground">{$t.ui.common.disabledText}</span>
			{/if}
		</label>
		<div class="flex items-center space-x-1">
			<MultiSelect
				options={(systemConfig?.network_interfaces || []).map((iface) => ({
					value: iface.name,
					label: iface.name,
					description: iface.ip
				}))}
				bind:selectedValues={selectedInterfaces}
				placeholder={systemConfig?.supports_interface_selection
					? $t.ui.common.selectInterfaces
					: $t.ui.common.interfaceSelectionDisabled}
				name="interface"
				disabled={disabled || !systemConfig?.supports_interface_selection}
			/>
			<Popover.Root>
				<Popover.Trigger>
					<Button variant="secondary" size="icon" type="button" class="shrink-0"
						><InfoIcon class="h-4 w-4" /></Button
					>
				</Popover.Trigger>
				<Popover.Content>
					<p class="mb-2 text-sm font-semibold">{$t.ui.host.info.interfaceTitle}</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.interfaceDescription}
					</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.interfaceEmpty}
					</p>
					{#if !systemConfig?.supports_interface_selection}
						<p class="mt-2 text-sm font-semibold text-orange-600">
							{$t.ui.host.info.interfaceDisabledTitle}
						</p>
						<p class="text-sm text-muted-foreground">
							{$t.ui.host.info.interfaceDisabledDescription}
						</p>
					{/if}
				</Popover.Content>
			</Popover.Root>
		</div>
	</div>

	<!-- Static IP Section with Fallback Checkbox -->
	<div class="flex w-full flex-col gap-2 rounded-md border p-3">
		<label for="static-ip" class="text-sm font-medium">{$t.ui.common.staticIpAddress}</label>
		<div class="flex items-center space-x-1">
			<Input
				id="static-ip"
				name="static_ip"
				bind:value={staticIp}
				placeholder={$t.ui.common.staticIpOptional}
				class="flex-1"
				{disabled}
			/>
			<Popover.Root>
				<Popover.Trigger>
					<Button variant="secondary" size="icon" type="button" class="shrink-0"
						><InfoIcon class="h-4 w-4" /></Button
					>
				</Popover.Trigger>
				<Popover.Content>
					<p class="mb-2 text-sm font-semibold">{$t.ui.host.info.staticIpTitle}</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.staticIpDescription}
					</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.staticIpUsage}
					</p>
					<p class="mb-1 text-sm font-semibold text-orange-600">
						{$t.ui.host.info.staticIpWarning}
					</p>
					<p class="text-sm text-muted-foreground">
						{$t.ui.host.info.staticIpWarningText}
					</p>
				</Popover.Content>
			</Popover.Root>
		</div>

		<!-- Fallback checkbox grouped with Static IP -->
		<div class="mt-2 flex items-start space-x-1">
			<div class="flex flex-1 items-center space-x-2">
				<input
					type="checkbox"
					id="use-fallback"
					bind:checked={useAsFallback}
					class="h-4 w-4 rounded border-gray-300"
					disabled={disabled || !staticIp}
				/>
				<label for="use-fallback" class="text-sm font-medium">
					{$t.ui.common.useAsFallbackOnly}
				</label>
			</div>
			<Popover.Root>
				<Popover.Trigger>
					<Button variant="secondary" size="icon" type="button" class="shrink-0">
						<InfoIcon class="h-4 w-4" />
					</Button>
				</Popover.Trigger>
				<Popover.Content>
					<p class="mb-2 text-sm font-semibold">{$t.ui.host.info.fallbackTitle}</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.fallbackChecked}
					</p>
					<p class="mb-1 text-sm">
						{$t.ui.host.info.fallbackUnchecked}
					</p>
					<p class="text-sm text-muted-foreground">
						{$t.ui.host.info.fallbackNote}
					</p>
				</Popover.Content>
			</Popover.Root>
		</div>
	</div>
</div>
