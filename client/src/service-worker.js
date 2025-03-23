/// <reference types="@sveltejs/kit" />
import { version } from '$service-worker';

console.log('service worker loaded');

self.addEventListener('pushsubscriptionchange', (event) => {
	console.log({ type: 'pushsubscriptionchange', version, event });
});

self.addEventListener('push', (event) => {
	const { body } = JSON.parse(event.data.text());
	console.log({ type: 'push', version, event, body });
	self.registration.showNotification(body, {
		body
	});
});
