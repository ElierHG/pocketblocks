import React, { useState, useRef, useEffect, useContext, useCallback } from "react";
import styled from "styled-components";
import { Button, Input, Spin, message } from "antd";
import { CloseOutlined, SendOutlined, RobotOutlined, SettingOutlined } from "@ant-design/icons";
import Api from "api/api";
import { EditorContext } from "comps/editorState";
import { useSelector } from "react-redux";
import { currentApplication } from "redux/selectors/applicationSelector";

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
  &:hover {
    background: rgba(255, 255, 255, 0.2);
  }
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

const ApplyButton = styled(Button)`
  margin-top: 6px;
  font-size: 12px;
`;

const InputArea = styled.div`
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
`;

const SetupArea = styled.div`
  padding: 20px 16px;
  text-align: center;
`;

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  dsl?: any;
  applied?: boolean;
}

interface AiChatPanelProps {
  visible: boolean;
  onClose: () => void;
}

export default function AiChatPanel({ visible, onClose }: AiChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [hasApiKey, setHasApiKey] = useState<boolean | null>(null);
  const [apiKeyInput, setApiKeyInput] = useState("");
  const [showSetup, setShowSetup] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const editorState = useContext(EditorContext);
  const application = useSelector(currentApplication);

  useEffect(() => {
    if (visible) {
      checkApiKey();
    }
  }, [visible]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const checkApiKey = async () => {
    try {
      const resp = await Api.get("ai/config");
      setHasApiKey(resp.data?.data?.hasApiKey ?? false);
    } catch {
      setHasApiKey(false);
    }
  };

  const saveApiKey = async () => {
    if (!apiKeyInput.trim()) return;
    try {
      await Api.put("ai/config", { apiKey: apiKeyInput.trim() });
      setHasApiKey(true);
      setShowSetup(false);
      setApiKeyInput("");
      message.success("API key saved");
    } catch {
      message.error("Failed to save API key");
    }
  };

  const getCurrentDSL = useCallback(() => {
    if (!editorState) return {};
    try {
      return editorState.rootComp.toJsonValue();
    } catch {
      return {};
    }
  }, [editorState]);

  const applyDSL = useCallback(
    (newDSL: any) => {
      if (!editorState || !newDSL) return;
      try {
        editorState.setComp((comp) => {
          return comp.reduce(
            comp.changeValueAction(newDSL)
          );
        });
        message.success("AI changes applied to canvas");
      } catch (e) {
        message.error("Failed to apply DSL changes");
        console.error("Apply DSL error:", e);
      }
    },
    [editorState]
  );

  const sendMessage = async () => {
    const msg = input.trim();
    if (!msg || loading) return;

    setInput("");
    const userMsg: ChatMessage = { role: "user", content: msg };
    setMessages((prev) => [...prev, userMsg]);
    setLoading(true);

    try {
      const currentDSL = getCurrentDSL();
      const resp = await Api.post("ai/chat", {
        message: msg,
        currentDSL,
      });

      const data = resp.data?.data;
      const explanation = data?.explanation || "Here are the changes.";
      const dsl = data?.dsl || null;

      const assistantMsg: ChatMessage = {
        role: "assistant",
        content: explanation,
        dsl,
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch (e: any) {
      const errMsg =
        e?.response?.data?.message || e?.message || "AI request failed";
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: `Error: ${errMsg}` },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleApply = (idx: number) => {
    const msg = messages[idx];
    if (msg?.dsl) {
      applyDSL(msg.dsl);
      setMessages((prev) =>
        prev.map((m, i) => (i === idx ? { ...m, applied: true } : m))
      );
    }
  };

  if (!visible) return null;

  return (
    <PanelOverlay>
      <PanelHeader>
        <HeaderLeft>
          <RobotOutlined style={{ fontSize: 18 }} />
          AI Assistant
        </HeaderLeft>
        <HeaderActions>
          <IconBtn onClick={() => setShowSetup(!showSetup)} title="Settings">
            <SettingOutlined style={{ fontSize: 14 }} />
          </IconBtn>
          <IconBtn onClick={onClose} title="Close">
            <CloseOutlined style={{ fontSize: 14 }} />
          </IconBtn>
        </HeaderActions>
      </PanelHeader>

      {showSetup || hasApiKey === false ? (
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
          <Button type="primary" onClick={saveApiKey} block>
            Save API Key
          </Button>
          {hasApiKey && (
            <Button
              type="link"
              onClick={() => setShowSetup(false)}
              style={{ marginTop: 8 }}
            >
              Back to chat
            </Button>
          )}
        </SetupArea>
      ) : (
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
                  {msg.content}
                </BubbleContent>
                {msg.role === "assistant" && msg.dsl && !msg.applied && (
                  <ApplyButton
                    type="primary"
                    size="small"
                    onClick={() => handleApply(idx)}
                  >
                    Apply to Canvas
                  </ApplyButton>
                )}
                {msg.applied && (
                  <span style={{ fontSize: 11, color: "#52c41a", marginTop: 4 }}>
                    Applied
                  </span>
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
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={sendMessage}
              loading={loading}
            />
          </InputArea>
        </>
      )}
    </PanelOverlay>
  );
}
