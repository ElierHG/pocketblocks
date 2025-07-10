import { AxiosPromise } from "axios";
import Api from "./api";
import { GenericApiResponse } from "./apiResponses";

export interface Connection {
  id: string;
  name: string;
  type: string;
  config: string;
  created: string;
  updated: string;
}

export class ConnectionApi extends Api {
  static url = "pbl/connections";

  static list(): AxiosPromise<GenericApiResponse<Connection[]>> {
    return Api.get(ConnectionApi.url);
  }

  static create(data: Partial<Connection>): AxiosPromise<GenericApiResponse<Connection>> {
    return Api.post(ConnectionApi.url, data);
  }

  static update(id: string, data: Partial<Connection>): AxiosPromise<GenericApiResponse<Connection>> {
    return Api.put(`${ConnectionApi.url}/${id}`, data);
  }

  static delete(id: string): AxiosPromise<GenericApiResponse<void>> {
    return Api.delete(`${ConnectionApi.url}/${id}`);
  }
}

export default ConnectionApi;
