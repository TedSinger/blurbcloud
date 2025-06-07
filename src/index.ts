import { DurableObject } from "cloudflare:workers";
import BlurbStuff from "./blurb";


export class BlurbServer extends DurableObject {
	subscribers: Map<string, Map<WritableStreamDefaultWriter<Object>, number>>;
	sql: SqlStorage;
	constructor(ctx: DurableObjectState, env: Env) {
		super(ctx, env);
		this.subscribers = new Map<string, Map<WritableStreamDefaultWriter<Object>, number>>();
		this.sql = ctx.storage.sql;
		this.sql.exec("CREATE TABLE IF NOT EXISTS blurbs (id TEXT, version INTEGER, body TEXT, PRIMARY KEY (id, version))");
	}

	async getBlurb(blurb_id: string): Promise<{blurb_text: string, blurb_version: number}> {
		const cursor = this.sql.exec("SELECT id, version, body FROM blurbs WHERE id = ? and version = (SELECT MAX(version) FROM blurbs WHERE id = ?)", blurb_id, blurb_id);
		for (const row of cursor) {
			return {
				blurb_text: row.body as string,
				blurb_version: row.version as number
			}
		}
		return {
			blurb_text: '<u><em style="background-color: rgb(255, 240, 201)">Blurb.cloud</em></u> is a shared, local billboard. Anyone who sees a blurb can change the blurb.',
			blurb_version: 0
		}
	}

	async putBlurb(blurb_id: string, blurb_text: string, blurb_version: number): Promise<void> {
		blurb_text = BlurbStuff.sanitize_user_supplied_html(blurb_text);
		const cursor = this.sql.exec("insert into blurbs (id, version, body) select ?, ?, ? where ? > (select coalesce(max(version), 0) from blurbs where id = ?)", blurb_id, blurb_version, blurb_text, blurb_version, blurb_id);

		if (cursor.rowsWritten > 0 && this.subscribers.has(blurb_id)) {
			const subscribers = this.subscribers.get(blurb_id) || new Map<WritableStreamDefaultWriter<Object>, number>();
			for (const writer of subscribers.keys()) {
				writer.write({
					blurb_text: blurb_text,
					blurb_version: blurb_version
				})
			}
			console.log("published to " + subscribers.size + " subscribers");
		}
	}
	
	async subscribe(blurb_id: string): Promise<ReadableStream<Uint8Array>> {
		const { readable, writable } = new TransformStream<Object, Uint8Array>({
			transform(chunk, controller) {
				controller.enqueue(new TextEncoder().encode("data: " + JSON.stringify(chunk) + "\n\n"));
			}
		});
		const writer = writable.getWriter();
		writer.write(await this.getBlurb(blurb_id));
		if (!this.subscribers.has(blurb_id)) {
			this.subscribers.set(blurb_id, new Map<WritableStreamDefaultWriter<Object>, number>());
		}
		this.subscribers.get(blurb_id)?.set(writer, 0);
		return readable;
	}
}

export default {
	async fetch(request, env, ctx): Promise<Response> {

		if (new URL(request.url).pathname === '/') {
			const characters = 'abcdefghijklmnopqrstuvwxyz';
			let code = '';
			for (let i = 0; i < 4; i++) {
				code += characters.charAt(Math.floor(Math.random() * characters.length));
			}
			return Response.redirect(`${request.url}blurb/${code}`, 302);
		} else {
			const blurb_id = new URL(request.url).pathname.split('/')[2];
			const dob_id = env.BLURB_SERVER.idFromName("blurb");
			const dob = env.BLURB_SERVER.get(dob_id);
			const blurb = await dob.getBlurb(blurb_id);

			if (new URL(request.url).pathname.startsWith('/blurb/')) {
				if (request.method === 'PUT') {
					const blurb_message = await request.text();
					const blurb_message_json = JSON.parse(blurb_message);
					const blurb_text = blurb_message_json.blurb_text;
					const blurb_version = blurb_message_json.blurb_version;

					await dob.putBlurb(blurb_id, blurb_text, blurb_version);
					return new Response(null, {
						status: 200
					});
				} else {
					const template = await env.ASSETS.fetch(new URL("/view.html", request.url));
					const html = await template.text();
					const resp = html.replace(/idgoeshere/g, blurb_id)
						.replace(/versiongoeshere/g, blurb.blurb_version.toString())
						.replace(/blurbtextgoeshere/g, blurb.blurb_text)
						.replace(/qrcodegoeshere/g, BlurbStuff.get_qr_code(request.url));

					return new Response(resp, {
							headers: {
								"Content-Type": "text/html"
							}
						});
				}
			} else if (new URL(request.url).pathname.startsWith('/raw/')) {
				return new Response(JSON.stringify({
					blurb_text: blurb.blurb_text,
					blurb_version: blurb.blurb_version
				}), {
					headers: {
						"Content-Type": "application/json"
					}
				});
			} else if (new URL(request.url).pathname.startsWith('/stream/')) {
				const response: ReadableStream<Uint8Array> = await dob.subscribe(blurb_id);
				return new Response(response, {
					headers: {
						"Content-Type": "text/event-stream"
					}
				});
			} else {
				return Response.redirect(new URL('/', request.url).toString(), 302);
			}
		}
	},
} satisfies ExportedHandler<Env>;
