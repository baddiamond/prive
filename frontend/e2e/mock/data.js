// Fictional data for the UI regression harness. Every name, number and message is invented
// (CLAUDE.md: never real phone numbers or message content).

const NOW = Date.now();
const MIN = 60_000;

const whatsapp = {
  id: "whatsapp",
  name: "WhatsApp",
  shortName: "WA",
  loginMethods: ["qr"],
  capabilities: {
    sendText: true,
    readReceipts: true,
    typing: false,
    media: false,
    reactions: false,
    edit: false,
    delete: false,
    replies: false,
    historyOnDemand: true,
  },
};

export const engines = [whatsapp];

const colors = ["green", "amber", "rose", "sky", "violet", "coral"];
const accountNames = ["Personal", "Work", "Family business", "Club", "Side project", "Volunteering"];

/** Builds the accounts of a scenario. */
export function makeAccounts(count, states = []) {
  return Array.from({ length: count }, (_, i) => ({
    account: {
      id: `acc-${i + 1}`,
      engineId: "whatsapp",
      remoteId: `100000000${String(i).padStart(3, "0")}@s.whatsapp.net`,
      displayName: i < accountNames.length ? accountNames[i] : `Account ${i + 1}`,
      color: colors[i % colors.length],
      position: i,
      createdAt: new Date(NOW - 86_400_000).toISOString(),
    },
    state: states[i] ?? "connected",
    code: states[i] === "error" ? "errors.account.connectFailed" : "",
    manifest: whatsapp,
  }));
}

const people = [
  "Lucía Romero 👑❤️", "Diego Herrera", "Ana Torres", "Sofía Méndez", "Mateo Ruiz", "Valentina Cruz",
  "Javier Ortega", "Camila Navarro", "Andrés Molina", "Paula Castillo", "Marta Gil", "Leo Ruiz",
  "Inés Vidal", "Tomás Prieto", "Elena Soler", "Bruno Lozano",
];
const groups = [
  "🎮⚔️🕹️ Weekend gamers club with a very long name", "Product team", "Climbing crew", "Family",
  "Book club 📚", "Neighbours — building B",
];

const lines = [
  "Good morning!", "Are we still on for later?", "Sure, sounds good", "I'll be there in 10 minutes",
  "Did you see the photos?", "Haha, yes", "Can you send me the address?", "Thanks a lot",
  "Let me check and get back to you", "That works for me", "On my way", "Running a bit late, sorry",
  "Perfect", "What time works for you?", "How about Thursday?", "See you soon",
  "This is a longer message to see how bubbles wrap when people write a few sentences in one go. " +
    "It should never overflow the conversation, and the time should stay at the bottom right.",
  "Line one\nLine two\nLine three, after two line breaks",
  "https://example.com/a/very/long/link/without/spaces/that/must/wrap/inside/the/bubble/and/never/overflow/the/log?query=1234567890",
  "😂😂😂",
];
const kinds = ["text", "text", "text", "text", "text", "text", "sticker", "text", "voice", "text", "image", "text"];

/** Builds the chats of the given accounts. */
export function makeChats(accounts, perAccount) {
  const chats = [];
  for (const a of accounts) {
    for (let i = 0; i < perAccount; i++) {
      const group = i % 4 === 1;
      const name = group ? groups[i % groups.length] : people[i % people.length];
      chats.push({
        accountId: a.account.id,
        id: `${a.account.id}-chat-${i + 1}`,
        name: i === 7 ? "" : name,
        kind: group ? "group" : "direct",
        unreadCount: [0, 3, 0, 150, 12, 0, 1, 0][i % 8],
        lastMessagePreview: i % 5 === 2 ? lines[16] : lines[i % lines.length],
        lastMessageAt: new Date(NOW - (i * 37 + 3) * MIN).toISOString(),
        lastMessageKind: kinds[i % kinds.length],
        lastMessageFromMe: i % 3 === 0,
        muted: i % 9 === 4,
        archived: false,
        pinned: i < 2,
      });
    }
  }
  return chats;
}

/** Builds a chat's full stored history, oldest first. */
export function makeHistory(chat, count) {
  const group = chat.kind === "group";
  const end = new Date(chat.lastMessageAt).getTime();
  const out = [];
  for (let i = 0; i < count; i++) {
    const age = count - 1 - i;
    const fromMe = age % 3 === 0;
    const sender = group ? people[(age % 5) + 10] : chat.name;
    out.push({
      accountId: chat.accountId,
      chatId: chat.id,
      id: `${chat.id}-m${i}`,
      senderId: fromMe ? "" : `contact:${sender}`,
      senderName: fromMe ? "" : sender,
      fromMe,
      timestamp: new Date(end - age * 41 * MIN).toISOString(),
      kind: kinds[age % kinds.length],
      text: kinds[age % kinds.length] === "text" ? lines[age % lines.length] : "",
      status: fromMe ? ["read", "delivered", "sent"][age % 3] : "",
    });
  }
  return out;
}
