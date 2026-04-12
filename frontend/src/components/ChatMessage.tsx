import { chat } from "../../wailsjs/go/models";
import ActionCard from "./ActionCard";

type Props = {
  message: chat.ChatMessage;
  proposal?: chat.ActionProposal | undefined;
  vaultName?: string | undefined;
  onProposalResolved?: ((outcome: "executed" | "denied" | "dry_run" | "failed") => void) | undefined;
};

export default function ChatMessage({ message, proposal, vaultName, onProposalResolved }: Props) {
  const isUser = message.role === "user";

  return (
    <div style={{
      display: "flex",
      flexDirection: "column",
      alignItems: isUser ? "flex-end" : "flex-start",
      marginBottom: "12px",
      maxWidth: "100%",
    }}>
      {/* Role label */}
      <div style={{
        fontSize: "10px",
        color: "var(--term-dim)",
        marginBottom: "3px",
        letterSpacing: "0.05em",
        textTransform: "uppercase",
      }}>
        {isUser ? "You" : "Vault AI"}
      </div>

      {/* Bubble */}
      <div style={{
        maxWidth: "88%",
        padding: "8px 12px",
        borderRadius: isUser ? "12px 12px 2px 12px" : "12px 12px 12px 2px",
        background: isUser ? "var(--term-accent)" : "var(--term-panel)",
        color: isUser ? "#000" : "var(--term-fg)",
        border: isUser ? "none" : "1px solid var(--term-border)",
        fontSize: "13px",
        lineHeight: "1.5",
        whiteSpace: "pre-wrap",
        wordBreak: "break-word",
      }}>
        {message.content || (message.role === "assistant" ? <em style={{ opacity: 0.5 }}>…</em> : "")}

        {/* Inline ActionCard for assistant messages with proposals */}
        {!isUser && proposal && (
          <ActionCard
            proposal={proposal}
            {...(vaultName !== undefined && { vaultName })}
            {...(onProposalResolved !== undefined && { onResolved: onProposalResolved })}
          />
        )}
      </div>
    </div>
  );
}
