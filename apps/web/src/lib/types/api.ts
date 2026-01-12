// API type definitions shared across the application

export interface User {
	id: string;
	name: string;
	readonly: boolean;
	is_superuser: boolean;
	created: string;
	updated: string;
}

export interface AppConfig {
	os: string;
	url_prefix: string;
	use_auth: boolean;
	readonly_mode?: boolean;
	network_interfaces?: NetworkInterface[];
	supports_interface_selection?: boolean;
	user?: User;
}

export interface AuthStatus {
	authenticated: boolean;
	auth_enabled: boolean;
	user?: User;
}

export interface NetworkInterface {
	name: string;
	ip: string;
}

export interface Host {
	id: string;
	name: string;
	mac: string;
	broadcast: string;
	interface?: string;
	static_ip?: string;
	use_as_fallback?: boolean;
	ip?: string;
	user: string | null;
	created: string;
	updated: string;
}

export interface PingResult {
	ping_success: boolean;
	arp_success: boolean;
	rate_limited?: boolean;
	server_unreachable?: boolean;
}

export interface BulkPingResult {
	host_id: string;
	host_name: string;
	ping_success: boolean;
	arp_success: boolean;
	server_unreachable?: boolean;
}

export interface APIError {
	error: string;
	message?: string;
	status?: string;
}
