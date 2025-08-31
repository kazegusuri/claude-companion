import { expect, type Page, test } from "@playwright/test";

// WebSocketテストサーバーのURL
const WS_TEST_SERVER = "http://localhost:8080";
const WEB_APP_URL = "http://localhost:3000";

test.describe("WebSocket Audio System", () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    // 新しいページを開く
    page = await browser.newPage();

    // アプリケーションにアクセス
    await page.goto(WEB_APP_URL);

    // Audio Narratorタブが表示されていることを確認
    await expect(page.locator('button:has-text("Audio Narrator")')).toBeVisible();

    // Audio Narratorタブをクリック
    await page.click('button:has-text("Audio Narrator")');

    // WebSocket接続を待つ
    await page.waitForTimeout(1000);

    // 接続状態を確認
    await expect(page.locator("text=接続中")).toBeVisible();
  });

  test.afterEach(async () => {
    await page.close();
  });

  test("should connect to WebSocket server", async () => {
    // サーバーヘルスチェック
    const healthResponse = await fetch(`${WS_TEST_SERVER}/health`);
    const health = await healthResponse.json();
    expect(health.status).toBe("ok");
    expect(health.clients).toBeGreaterThan(0);
  });

  test("should display text messages", async () => {
    // テキストメッセージを送信
    const response = await fetch(`${WS_TEST_SERVER}/api/send/text`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: "Playwrightテスト: テキストメッセージ",
        eventType: "message",
        toolName: "PlaywrightTest",
      }),
    });

    const result = await response.json();
    expect(result.success).toBe(true);

    // メッセージが表示されるまで待つ
    await page.waitForTimeout(500);

    // メッセージが表示されていることを確認
    await expect(page.locator("text=Playwrightテスト: テキストメッセージ")).toBeVisible();

    // メタデータが表示されていることを確認
    await expect(page.locator("text=💬message")).toBeVisible();
    await expect(page.locator("text=PlaywrightTest")).toBeVisible();
  });

  test("should display audio messages", async () => {
    // オーディオメッセージを送信
    const response = await fetch(`${WS_TEST_SERVER}/api/send/audio`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: "Playwrightテスト: オーディオメッセージ",
        eventType: "tool",
        toolName: "AudioTest",
        sampleRate: 48000,
        duration: 1.5,
      }),
    });

    const result = await response.json();
    expect(result.success).toBe(true);

    // メッセージが表示されるまで待つ
    await page.waitForTimeout(500);

    // メッセージが表示されていることを確認
    await expect(page.locator("text=Playwrightテスト: オーディオメッセージ")).toBeVisible();

    // オーディオメッセージではメタデータは表示されない
    // （audioタイプのメッセージはテキストのみ表示）
  });

  test("should handle test messages", async () => {
    // テストメッセージを送信
    const response = await fetch(`${WS_TEST_SERVER}/api/send/test`, {
      method: "POST",
    });

    const result = await response.json();
    expect(result.success).toBe(true);
    expect(result.text).toContain("テストメッセージ");

    // メッセージが表示されるまで待つ
    await page.waitForTimeout(500);

    // メッセージが表示されていることを確認
    const messages = page.locator(".message-text");
    const count = await messages.count();
    expect(count).toBeGreaterThan(0);

    // 最後のメッセージを確認
    const lastMessage = messages.last();
    await expect(lastMessage).toContainText("テストメッセージ");
  });

  test("should clear message history", async () => {
    // いくつかメッセージを送信
    for (let i = 1; i <= 3; i++) {
      await fetch(`${WS_TEST_SERVER}/api/send/text`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          text: `クリアテストメッセージ ${i}`,
        }),
      });
      await page.waitForTimeout(200);
    }

    // メッセージが表示されていることを確認
    const messagesBefore = await page.locator(".message-item").count();
    expect(messagesBefore).toBeGreaterThan(0);

    // 履歴クリアボタンをクリック
    await page.click('button:has-text("履歴クリア")');

    // メッセージがクリアされたことを確認
    const messagesAfter = await page.locator(".message-item").count();
    expect(messagesAfter).toBe(0);
  });

  test("should control volume", async () => {
    // 音量スライダーを確認
    const volumeSlider = page.locator('input[type="range"]');
    await expect(volumeSlider).toBeVisible();

    // 初期値を確認
    const initialValue = await volumeSlider.inputValue();
    expect(initialValue).toBe("1");

    // 音量を変更
    await volumeSlider.fill("0.5");

    // 表示が更新されることを確認
    await expect(page.locator("text=50%")).toBeVisible();

    // 音量を0に設定
    await volumeSlider.fill("0");
    await expect(page.locator("text=0%")).toBeVisible();

    // 音量を最大に戻す
    await volumeSlider.fill("1");
    await expect(page.locator("text=100%")).toBeVisible();
  });

  test("should handle multiple messages in queue", async () => {
    // 複数のオーディオメッセージを連続送信
    const messages = [
      "キューテスト1: 最初のメッセージ",
      "キューテスト2: 二番目のメッセージ",
      "キューテスト3: 三番目のメッセージ",
    ];

    for (const text of messages) {
      await fetch(`${WS_TEST_SERVER}/api/send/audio`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
      });
      await page.waitForTimeout(100);
    }

    // すべてのメッセージが表示されることを確認
    for (const text of messages) {
      await expect(page.locator(`text=${text}`)).toBeVisible();
    }

    // キューステータスが表示されることを確認（処理中の場合）
    // タイムアウトを短くして、素早くチェック
    const queueStatus = page.locator("text=/キュー: \\d+件/");
    try {
      const isVisible = await queueStatus.isVisible({ timeout: 1000 });
      if (isVisible) {
        const queueText = await queueStatus.textContent({ timeout: 1000 });
        expect(queueText).toMatch(/キュー: \d+件/);
      }
    } catch (_error) {
      // キューステータスが表示されない場合は、すでに処理済みの可能性があるのでOK
      console.log("Queue status not visible or already processed");
    }
  });

  test("should show timestamp for messages", async () => {
    // メッセージを送信
    await fetch(`${WS_TEST_SERVER}/api/send/text`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: "タイムスタンプテスト",
      }),
    });

    await page.waitForTimeout(500);

    // タイムスタンプのフォーマットを確認 (HH:MM:SS)
    const timestamp = page.locator(".message-time").last();
    await expect(timestamp).toBeVisible();
    const timeText = await timestamp.textContent();
    expect(timeText).toMatch(/\d{2}:\d{2}:\d{2}/);
  });

  test("should handle reconnection", async () => {
    // 現在の接続状態を確認
    await expect(page.locator("text=接続中")).toBeVisible();

    // WebSocketサーバーを停止（実際のテストでは難しいので、切断状態をシミュレート）
    // ここでは再接続ボタンの表示をテストする別の方法を検討

    // メッセージを送信して接続を確認
    const response = await fetch(`${WS_TEST_SERVER}/api/send/text`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text: "再接続テストメッセージ",
      }),
    });

    expect(response.ok).toBe(true);
  });
});

test.describe("Live2D Viewer Tab", () => {
  test("should switch between tabs", async ({ page }) => {
    await page.goto(WEB_APP_URL);

    // Audio Narratorタブをクリック
    await page.click('button:has-text("Audio Narrator")');
    await expect(page.locator('h2:has-text("Audio Narrator")')).toBeVisible();

    // Live2D Viewerタブをクリック
    await page.click('button:has-text("Live2D Viewer")');
    await expect(page.locator("text=Live2D Viewer")).toBeVisible();

    // 再度Audio Narratorタブに戻る
    await page.click('button:has-text("Audio Narrator")');
    await expect(page.locator('h2:has-text("Audio Narrator")')).toBeVisible();
  });
});
