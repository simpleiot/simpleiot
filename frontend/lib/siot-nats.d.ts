// Types for siot-nats.js, the Simple IoT client for the browser.

import type { ConnectionOptions, Msg, NatsConnection, Status, Subscription } from "nats.ws"
import type { Node, Point, PointInput } from "./codec.js"

export type { Node, Point, PointInput } from "./codec.js"
export { DataType, decodeNodes, decodePoints, encodePoints } from "./codec.js"
export { DebugEvents, ErrorCode, Events } from "nats.ws"

export declare function userIdFromToken(token: string): string
export declare function inboxPrefix(userId: string): string

export interface ConnectOptions extends Partial<ConnectionOptions> {
	// url is the WebSocket URL of the Simple IoT HTTP server
	url?: string
	// token is the JWT from sign-in
	token?: string
	// user defaults to the ID in the token
	user?: string
}

export declare function connect(opts?: ConnectOptions): Promise<SiotConnection>

export interface Me {
	userId: string
	// anchors are the nodes the user sits under
	anchors: string[]
	// nodes are the user's node at each anchor
	nodes: Node[]
}

export interface GetNodesOptions {
	depth?: number
	type?: string
	includeDel?: boolean
	timeout?: number
}

// Message is a decoded message from a point subject. parentId is set for
// edge points only.
export interface Message {
	subject: string
	anchor?: string
	nodeId: string
	parentId?: string
	type: string
	key: string
	points: Point[]
}

// SiotSubscription is a nats.ws subscription that yields decoded messages.
export interface SiotSubscription extends AsyncIterable<Message> {
	unsubscribe(max?: number): void
	drain(): Promise<void>
	isClosed(): boolean
	getSubject(): string
	closed: Subscription["closed"]
}

export declare class SiotConnection {
	readonly nc: NatsConnection
	readonly userId: string
	readonly token: string
	status(): AsyncIterable<Status>
	closed(): Promise<void | Error>
	close(): Promise<void>
	request(subject: string, payload?: Uint8Array, timeout?: number): Promise<Msg>
	me(): Promise<Me>
	getNodes(anchor: string, parent: string, id: string, opts?: GetNodesOptions): Promise<Node[]>
	sendNodePoints(
		anchor: string,
		id: string,
		points: readonly PointInput[],
		opts?: { timeout?: number }
	): Promise<void>
	sendEdgePoints(
		anchor: string,
		id: string,
		parent: string,
		points: readonly PointInput[],
		opts?: { timeout?: number }
	): Promise<void>
	subscribe(subject: string): SiotSubscription
}
