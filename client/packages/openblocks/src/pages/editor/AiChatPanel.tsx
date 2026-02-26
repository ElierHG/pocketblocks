import React, { useState, useRef, useEffect, useContext, useCallback } from "react";
import styled from "styled-components";
import { Button, Input, Spin, message, Divider } from "antd";
import { CloseOutlined, SendOutlined, RobotOutlined, SettingOutlined, KeyOutlined, LinkOutlined, CheckCircleOutlined, UndoOutlined } from "@ant-design/icons";
import Api from "api/api";
import { EditorContext } from "comps/editorState";
import { useSelector } from "react-redux";
import { currentApplication } from "redux/selectors/applicationSelector";
import { SERVER_HOST } from "constants/apiConstants";
import _ from "lodash";
import {
  changeValueAction,
  multiChangeAction,
  wrapActionExtraInfo,
} from "openblocks-core";
import { addMapChildAction } from "comps/generators/sameTypeMap";
import { SimpleContainerComp } from "comps/comps/containerBase/simpleContainerComp";
import { genRandomKey } from "comps/utils/idGenerator";
import html2canvas from "html2canvas";

const PanelOverlay = styled.div`
  position: fixed;
  right: 16px;
  bottom: 16px;
  width: 420px;
  max-height: 600px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
  display: flex;
  flex-direction: column;
  z-index: 1000;
  overflow: hidden;
`;

const PanelHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, #315efb 0%, #5b8def 100%);
  color: #fff;
  font-weight: 600;
  font-size: 14px;
`;

const HeaderLeft = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const HeaderActions = styled.div`
  display: flex;
  gap: 4px;
`;

const IconBtn = styled.button`
  background: none;
  border: none;
  color: #fff;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  &:hover { background: rgba(255, 255, 255, 0.2); }
`;

const MessagesArea = styled.div`
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  max-height: 420px;
  min-height: 200px;
`;

const MessageBubble = styled.div<{ isUser: boolean }>`
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  align-items: ${(p) => (p.isUser ? "flex-end" : "flex-start")};
`;

const BubbleContent = styled.div<{ isUser: boolean }>`
  max-width: 85%;
  padding: 8px 12px;
  border-radius: ${(p) => (p.isUser ? "12px 12px 2px 12px" : "12px 12px 12px 2px")};
  background: ${(p) => (p.isUser ? "#315efb" : "#f0f2f5")};
  color: ${(p) => (p.isUser ? "#fff" : "#333")};
  font-size: 13px;
  line-height: 1.5;
  word-wrap: break-word;
  white-space: pre-wrap;
`;

const AppliedBadge = styled.div`
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 11px;
  color: #52c41a;
`;


const InputArea = styled.div`
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
`;

const SetupArea = styled.div`
  padding: 20px 16px;
  overflow-y: auto;
  max-height: 500px;
`;

const DeviceCodeBox = styled.div`
  background: #f5f7ff;
  border: 1px solid #d6e0ff;
  border-radius: 8px;
  padding: 16px;
  text-align: center;
  margin: 12px 0;
`;

const CodeDisplay = styled.div`
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 4px;
  color: #315efb;
  margin: 8px 0;
  font-family: monospace;
`;

const AuthMethodBtn = styled(Button)`
  width: 100%;
  height: 42px;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
`;

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  applied?: boolean;
}

interface AiChatPanelProps {
  visible: boolean;
  onClose: () => void;
}

type AuthView = "menu" | "apikey" | "device" | "chat";

export default function AiChatPanel({ visible, onClose }: AiChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [authConfigured, setAuthConfigured] = useState<boolean | null>(null);
  const [authView, setAuthView] = useState<AuthView>("menu");
  const [apiKeyInput, setApiKeyInput] = useState("");
  const [deviceCode, setDeviceCode] = useState<any>(null);
  const [polling, setPolling] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const pollTimerRef = useRef<any>(null);
  const editorState = useContext(EditorContext);
  const application = useSelector(currentApplication);

  useEffect(() => {
    if (visible) {
      checkAuth();
    }
    return () => {
      if (pollTimerRef.current) clearInterval(pollTimerRef.current);
    };
  }, [visible]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const checkAuth = async () => {
    try {
      const resp = await Api.get("ai/config");
      const data = resp.data?.data;
      const hasAuth = data?.hasApiKey || data?.hasCodexAuth;
      setAuthConfigured(hasAuth);
      if (hasAuth && !showSettings) {
        setAuthView("chat");
      } else {
        setAuthView("menu");
      }
    } catch {
      setAuthConfigured(false);
      setAuthView("menu");
    }
  };

  const saveApiKey = async () => {
    if (!apiKeyInput.trim()) return;
    try {
      await Api.put("ai/config", { apiKey: apiKeyInput.trim() });
      setAuthConfigured(true);
      setAuthView("chat");
      setApiKeyInput("");
      setShowSettings(false);
      message.success("API key saved");
    } catch {
      message.error("Failed to save API key");
    }
  };

  const CODEX_CLIENT_ID = "app_EMoamEEZ73f0CkXaXp7hrann";
  const DEVICE_CODE_URL = "https://auth0.openai.com/oauth/device/code";
  const TOKEN_URL = "https://auth0.openai.com/oauth/token";

  const startDeviceCode = async () => {
    try {
      const params = new URLSearchParams();
      params.set("client_id", CODEX_CLIENT_ID);
      params.set("scope", "openid profile email offline_access");
      params.set("audience", "https://api.openai.com/v1");

      const resp = await fetch(DEVICE_CODE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: params.toString(),
      });

      if (!resp.ok) {
        const text = await resp.text();
        throw new Error("Failed: " + text.slice(0, 200));
      }

      const data = await resp.json();
      if (data.user_code) {
        const dcData = {
          deviceCode: data.device_code,
          userCode: data.user_code,
          verificationUrl: data.verification_uri_complete || "https://auth.openai.com/codex/device",
          expiresIn: data.expires_in,
          interval: data.interval,
        };
        setDeviceCode(dcData);
        setAuthView("device");
        startPolling(dcData.deviceCode, dcData.interval || 5);
      } else {
        message.error("Failed to start sign-in flow");
      }
    } catch (e: any) {
      message.error(e?.message || "Failed to start sign-in flow");
    }
  };

  const startPolling = (dc: string, interval: number) => {
    setPolling(true);
    const pollInterval = Math.max(interval, 5) * 1000;
    pollTimerRef.current = setInterval(async () => {
      try {
        const params = new URLSearchParams();
        params.set("grant_type", "urn:ietf:params:oauth:grant-type:device_code");
        params.set("device_code", dc);
        params.set("client_id", CODEX_CLIENT_ID);

        const resp = await fetch(TOKEN_URL, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: params.toString(),
        });

        const data = await resp.json();

        if (data.access_token) {
          clearInterval(pollTimerRef.current);
          setPolling(false);
          // Save tokens to backend
          await Api.post("ai/auth/save-tokens", {
            accessToken: data.access_token,
            refreshToken: data.refresh_token || "",
          });
          setAuthConfigured(true);
          setAuthView("chat");
          setShowSettings(false);
          setDeviceCode(null);
          message.success("Signed in with ChatGPT!");
        } else if (data.error === "expired_token" || data.error === "access_denied") {
          clearInterval(pollTimerRef.current);
          setPolling(false);
          message.error(data.error_description || "Sign-in failed or expired");
        }
        // "authorization_pending" and "slow_down" keep polling
      } catch {
        // keep polling on network errors
      }
    }, pollInterval);
  };

  const importCodex = async () => {
    try {
      const resp = await Api.post("ai/auth/codex-import", {});
      const data = resp.data?.data;
      if (data?.method) {
        setAuthConfigured(true);
        setAuthView("chat");
        setShowSettings(false);
        message.success("Imported Codex CLI credentials (" + data.method + ")");
      }
    } catch (e: any) {
      message.error(e?.response?.data?.message || "No Codex CLI credentials found");
    }
  };

  const executeAction = useCallback((action: string, params: any) => {
    if (!editorState) return;
    try {
      if (action === "add_component") {
        const { comp_type, name, props, x, y, w, h } = params;
        const uiComp = editorState.getUIComp().getComp();
        if (!uiComp) return;

        const container = uiComp.realSimpleContainer() as SimpleContainerComp;
        if (!container) return;

        const key = genRandomKey();
        const compName = name || editorState.getNameGenerator().genItemName(comp_type);
        const currentLayout = container.children.layout.getView();

        const layoutItem = {
          i: key,
          x: x ?? 0,
          y: y ?? Object.keys(currentLayout).length * 5,
          w: w ?? 12,
          h: h ?? 5,
        };

        container.dispatch(
          wrapActionExtraInfo(
            multiChangeAction({
              layout: changeValueAction({ ...currentLayout, [key]: layoutItem }, true),
              items: addMapChildAction(key, {
                compType: comp_type,
                name: compName,
                comp: props || undefined,
              }),
            }),
            { compInfos: [{ compName, compType: comp_type, type: "add" }] }
          )
        );
      }
    } catch (e) {
      console.error("Action execution error:", e);
    }
  }, [editorState]);

  const MAX_REVIEW_ROUNDS = 2;

  const captureCanvas = async (): Promise<string | null> => {
    try {
      const editorContainer = document.querySelector("[class*='EditorContainerWithViewMode']");
      if (!editorContainer) return null;
      const canvas = await html2canvas(editorContainer as HTMLElement, {
        scale: 0.5,
        useCORS: true,
        logging: false,
        backgroundColor: "#ffffff",
      });
      return canvas.toDataURL("image/jpeg", 0.6).split(",")[1];
    } catch {
      return null;
    }
  };

  const streamOnce = async (
    msg: string,
    screenshot: string | null,
    assistantIdx: { current: number }
  ): Promise<{ actionCount: number; explanation: string; error?: string }> => {
    const baseURL = `${_.trimEnd(SERVER_HOST, "/")}/api/ai/chat/stream`;
    const resp = await fetch(baseURL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ message: msg, screenshot }),
    });
    if (!resp.ok || !resp.body) throw new Error("Stream request failed");

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let actionCount = 0;
    let explanation = "";

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (!line.startsWith("data: ")) continue;
        let event: any;
        try { event = JSON.parse(line.slice(6)); } catch { continue; }

        if (event.type === "delta") {
          const idx = assistantIdx.current;
          setMessages((prev) =>
            prev.map((m, i) => (i === idx ? { ...m, content: m.content + event.data } : m))
          );
        } else if (event.type === "action") {
          executeAction(event.action, event.params);
          actionCount++;
          const idx = assistantIdx.current;
          const label = event.params?.name || event.params?.comp_type || "";
          setMessages((prev) =>
            prev.map((m, i) =>
              i === idx ? { ...m, content: m.content + `\n+ ${event.action}: ${label}` } : m
            )
          );
        } else if (event.type === "done") {
          explanation = event.explanation || "";
        } else if (event.type === "error") {
          return { actionCount, explanation, error: event.data };
        }
      }
    }
    return { actionCount, explanation };
  };

  const sendMessage = async () => {
    const msg = input.trim();
    if (!msg || loading) return;
    setInput("");
    setMessages((prev) => [...prev, { role: "user", content: msg }]);
    setLoading(true);

    const assistantIdx = { current: -1 };

    try {
      const screenshot = await captureCanvas();

      setMessages((prev) => {
        assistantIdx.current = prev.length;
        return [...prev, { role: "assistant" as const, content: "", applied: false }];
      });

      const result = await streamOnce(msg, screenshot, assistantIdx);

      if (result.error) {
        const idx = assistantIdx.current;
        setMessages((prev) =>
          prev.map((m, i) => (i === idx ? { ...m, content: `Error: ${result.error}` } : m))
        );
        setLoading(false);
        return;
      }

      const idx = assistantIdx.current;
      setMessages((prev) =>
        prev.map((m, i) =>
          i === idx
            ? { ...m, content: result.explanation || m.content || "Done.", applied: result.actionCount > 0 }
            : m
        )
      );

      if (result.actionCount > 0) {
        for (let round = 0; round < MAX_REVIEW_ROUNDS; round++) {
          await new Promise((r) => setTimeout(r, 800));
          const reviewScreenshot = await captureCanvas();
          if (!reviewScreenshot) break;

          setMessages((prev) => {
            assistantIdx.current = prev.length;
            return [...prev, { role: "assistant" as const, content: "Reviewing layout..." }];
          });

          const rr = await streamOnce(
            "REVIEW: Screenshot shows the canvas after your changes. Check sizing, spacing, overlap. Fix issues with modify_page or confirm it looks good.",
            reviewScreenshot,
            assistantIdx
          );

          if (rr.error) break;

          const rIdx = assistantIdx.current;
          setMessages((prev) =>
            prev.map((m, i) =>
              i === rIdx
                ? { ...m, content: rr.explanation || m.content || "Looks good!", applied: rr.actionCount > 0 }
                : m
            )
          );

          if (rr.actionCount === 0) break;
        }
      }
    } catch (e: any) {
      if (assistantIdx.current >= 0) {
        const idx = assistantIdx.current;
        setMessages((prev) =>
          prev.map((m, i) =>
            i === idx && !m.content ? { ...m, content: `Error: ${e?.message || "Failed"}` } : m
          )
        );
      } else {
        setMessages((prev) => [...prev, {
          role: "assistant" as const, content: `Error: ${e?.message || "Failed"}`,
        }]);
      }
    } finally {
      setLoading(false);
    }
  };

  if (!visible) return null;

  const renderAuthMenu = () => (
    <SetupArea>
      <p style={{ marginBottom: 16, color: "#666", textAlign: "center" }}>
        Connect to OpenAI to enable AI-powered page building.
      </p>

      <AuthMethodBtn type="primary" onClick={startDeviceCode}>
        <LinkOutlined />
        Sign in with ChatGPT
      </AuthMethodBtn>

      <AuthMethodBtn onClick={() => setAuthView("apikey")}>
        <KeyOutlined />
        Enter API Key
      </AuthMethodBtn>

      <AuthMethodBtn onClick={importCodex} type="dashed">
        Import from Codex CLI
      </AuthMethodBtn>

      {authConfigured && (
        <>
          <Button type="link" onClick={() => { setAuthView("chat"); setShowSettings(false); }} block style={{ marginTop: 8 }}>
            Back to chat
          </Button>
          <Button type="link" danger onClick={async () => {
            try {
              await Api.put("ai/config", { clear: true });
              setAuthConfigured(false);
              message.success("AI auth cleared");
            } catch { message.error("Failed to clear"); }
          }} block size="small" style={{ marginTop: 4, fontSize: 12 }}>
            Disconnect
          </Button>
        </>
      )}
    </SetupArea>
  );

  const renderApiKeyForm = () => (
    <SetupArea>
      <p style={{ marginBottom: 12, color: "#666" }}>
        Enter your OpenAI API key to enable AI features.
      </p>
      <Input.Password
        placeholder="sk-..."
        value={apiKeyInput}
        onChange={(e) => setApiKeyInput(e.target.value)}
        onPressEnter={saveApiKey}
        style={{ marginBottom: 12 }}
      />
      <Button type="primary" onClick={saveApiKey} block>Save API Key</Button>
      <Button type="link" onClick={() => setAuthView("menu")} block style={{ marginTop: 8 }}>
        Back
      </Button>
    </SetupArea>
  );

  const renderDeviceCode = () => (
    <SetupArea>
      {deviceCode ? (
        <>
          <p style={{ color: "#333", marginBottom: 4 }}>
            Open this link and sign in:
          </p>
          <DeviceCodeBox>
            <a
              href={deviceCode.verificationUrl}
              target="_blank"
              rel="noopener noreferrer"
              style={{ color: "#315efb", fontWeight: 600, fontSize: 14 }}
            >
              {deviceCode.verificationUrl}
            </a>
            <Divider style={{ margin: "12px 0" }} />
            <p style={{ color: "#666", margin: 0, fontSize: 12 }}>Enter this code:</p>
            <CodeDisplay>{deviceCode.userCode}</CodeDisplay>
          </DeviceCodeBox>
          {polling && (
            <div style={{ textAlign: "center", color: "#999", fontSize: 12 }}>
              <Spin size="small" /> Waiting for authorization...
            </div>
          )}
        </>
      ) : (
        <div style={{ textAlign: "center" }}>
          <Spin /> Starting sign-in flow...
        </div>
      )}
      <Button
        type="link"
        onClick={() => {
          if (pollTimerRef.current) clearInterval(pollTimerRef.current);
          setPolling(false);
          setDeviceCode(null);
          setAuthView("menu");
        }}
        block
        style={{ marginTop: 12 }}
      >
        Cancel
      </Button>
    </SetupArea>
  );

  const renderChat = () => (
    <>
      <MessagesArea>
        {messages.length === 0 && (
          <div style={{ textAlign: "center", color: "#999", padding: "40px 0" }}>
            <RobotOutlined style={{ fontSize: 32, marginBottom: 12, display: "block" }} />
            Ask me to help build your page!
            <br />
            <span style={{ fontSize: 12 }}>
              e.g. "Add a table with sample user data" or "Create a dashboard with charts"
            </span>
          </div>
        )}
        {messages.map((msg, idx) => (
          <MessageBubble key={idx} isUser={msg.role === "user"}>
            <BubbleContent isUser={msg.role === "user"}>
              {msg.content || (loading && idx === messages.length - 1 ? "" : "...")}
            </BubbleContent>
            {msg.role === "assistant" && msg.applied && (
              <AppliedBadge>
                <CheckCircleOutlined /> Applied to canvas
              </AppliedBadge>
            )}
          </MessageBubble>
        ))}
        {loading && (
          <div style={{ textAlign: "center", padding: "12px 0" }}>
            <Spin size="small" /> <span style={{ color: "#999", marginLeft: 8 }}>Thinking...</span>
          </div>
        )}
        <div ref={messagesEndRef} />
      </MessagesArea>
      <InputArea>
        <Input
          placeholder="Describe what you want to build..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onPressEnter={sendMessage}
          disabled={loading}
          style={{ flex: 1 }}
        />
        <Button type="primary" icon={<SendOutlined />} onClick={sendMessage} loading={loading} />
      </InputArea>
    </>
  );

  return (
    <PanelOverlay>
      <PanelHeader>
        <HeaderLeft>
          <RobotOutlined style={{ fontSize: 18 }} />
          AI Assistant
        </HeaderLeft>
        <HeaderActions>
          <IconBtn onClick={() => {
            setShowSettings(true);
            setAuthView("menu");
          }} title="Settings">
            <SettingOutlined style={{ fontSize: 14 }} />
          </IconBtn>
          <IconBtn onClick={onClose} title="Close">
            <CloseOutlined style={{ fontSize: 14 }} />
          </IconBtn>
        </HeaderActions>
      </PanelHeader>

      {authView === "menu" && renderAuthMenu()}
      {authView === "apikey" && renderApiKeyForm()}
      {authView === "device" && renderDeviceCode()}
      {authView === "chat" && renderChat()}
    </PanelOverlay>
  );
}
