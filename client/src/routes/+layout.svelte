<script lang="ts">
	import '../app.css';
	import { env } from '$env/dynamic/public';

	// Function to convert base64 string to Uint8Array for applicationServerKey
	function urlBase64ToUint8Array(base64String: string) {
		const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
		const base64 = (base64String + padding).replace(/\-/g, '+').replace(/_/g, '/');

		const rawData = window.atob(base64);
		const outputArray = new Uint8Array(rawData.length);

		for (let i = 0; i < rawData.length; ++i) {
			outputArray[i] = rawData.charCodeAt(i);
		}
		return outputArray;
	}

	if (navigator.serviceWorker) {
		navigator.serviceWorker.ready.then((r) => {
			r.pushManager
				.getSubscription()
				.then((s) => {
					if (s) {
						return s;
					}

					const applicationServerKey = urlBase64ToUint8Array(env.PUBLIC_VAPID_PUBLIC_KEY);

					if (r.active) {
						r.active.postMessage({
							type: 'storeApplicationServerKey',
							key: applicationServerKey
						});
					}

					return r.pushManager.subscribe({
						userVisibleOnly: true,
						applicationServerKey
					});
				})
				.then((s) => {
					fetch('http://localhost:8080/subscriptions', {
						method: 'POST',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify(s)
					}).catch((err) => {
						console.error('Failed to register subscription:', err);
					});
				});
		});
	}

	let { children } = $props();
</script>

{@render children()}
