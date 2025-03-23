<script lang="ts">
	// You can now use showNotification
	async function enablePushNotifications() {
		const permission = await window.Notification.requestPermission();
		if (permission === 'granted') {
			console.log('Notification permission granted');
		}
	}

	async function sendNotification() {
		if (navigator.serviceWorker) {
			navigator.serviceWorker.ready.then(async (r) => {
				const permission = window.Notification.permission;
				if (permission === 'granted') {
					await r.showNotification('Test Notification', {
						body: 'This is a test notification',
						dir: 'auto',
						requireInteraction: true
					});
				}
			});
		}
	}
</script>

<h1>Welcome to SvelteKit</h1>
<p>Visit <a href="https://svelte.dev/docs/kit">svelte.dev/docs/kit</a> to read the documentation</p>

<button onclick={enablePushNotifications}>Enable notifications</button>

<button onclick={sendNotification}>Send notification</button>
