import { connections } from "@/api";
import { mocker } from "@/mocker";
import { adminRoute, createDefaultErrorResponse, createDefaultResponse } from "@/utils";

export default [
  mocker.get("/api/pbl/connections", adminRoute(async () => {
    const resp = await connections.list();
    return resp.data ? createDefaultResponse(resp.data) : createDefaultErrorResponse([resp]);
  })),
  mocker.post("/api/pbl/connections", adminRoute(async (req) => {
    const resp = await connections.create(req.config.data);
    return resp.data ? createDefaultResponse(resp.data) : createDefaultErrorResponse([resp]);
  })),
  mocker.patch("/api/pbl/connections/:id", adminRoute(async (req) => {
    const resp = await connections.update({ id: req.params.id as string, ...req.config.data });
    return resp.data ? createDefaultResponse(resp.data) : createDefaultErrorResponse([resp]);
  })),
  mocker.delete("/api/pbl/connections/:id", adminRoute(async (req) => {
    const resp = await connections.remove(req.params.id as string);
    return resp.status === 200 ? createDefaultResponse() : createDefaultErrorResponse([resp]);
  })),
];
