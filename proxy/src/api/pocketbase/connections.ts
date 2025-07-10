import { Connection, APIResponse } from "@/types";
import { pbl, createDefaultErrorResponse } from "./utils";

export async function list(): APIResponse<Connection[]> {
  try {
    const conns = await pbl.connections.getFullList({ sort: "-updated,-created" });
    return { status: 200, data: conns };
  } catch (e) {
    return createDefaultErrorResponse(e);
  }
}

export async function create(params: Partial<Connection>): APIResponse<Connection> {
  try {
    const conn = await pbl.connections.create(params);
    return { status: 200, data: conn };
  } catch (e) {
    return createDefaultErrorResponse(e);
  }
}

export async function update(params: Partial<Connection> & { id: string }): APIResponse<Connection> {
  try {
    const conn = await pbl.connections.update(params.id, params);
    return { status: 200, data: conn };
  } catch (e) {
    return createDefaultErrorResponse(e);
  }
}

export async function remove(id: string): APIResponse {
  try {
    await pbl.connections.delete(id);
    return { status: 200 };
  } catch (e) {
    return createDefaultErrorResponse(e);
  }
}
