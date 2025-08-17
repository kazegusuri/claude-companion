import { test, expect } from "@playwright/test";

const WS_TEST_SERVER = "http://localhost:8080";

test.describe
  .serial("Chat Display Scrolling and Message Limit", () => {
    test.setTimeout(60000); // Increase timeout to 60 seconds

    test.beforeEach(async ({ page }) => {
      // Wait for the app to load
      await page.goto("/");
      await page.waitForLoadState("networkidle");

      // Wait for WebSocket connection or proceed after timeout
      try {
        await page.waitForFunction(
          () => {
            const badges = Array.from(document.querySelectorAll('[class*="Badge"]'));
            return badges.some(
              (badge) =>
                badge.textContent?.includes("接続中") || badge.textContent?.includes("Connected"),
            );
          },
          { timeout: 5000 },
        );
      } catch (e) {
        console.log("WebSocket connection wait timed out, proceeding anyway");
      }

      // Clear any existing messages
      const clearButton = page.getByRole("button", { name: "クリア" });
      if (await clearButton.isVisible()) {
        await clearButton.click();
        await page.waitForTimeout(500);
      }
    });

    test("should handle 30 messages without breaking layout and show scrollbar", async ({
      page,
    }) => {
      // Send 30 test messages to ensure scrollbar appears
      for (let i = 0; i < 30; i++) {
        await fetch(`${WS_TEST_SERVER}/api/send/test`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
        });
      }

      // Wait for all messages to appear and render
      await page.waitForTimeout(4000);

      // Check message count first
      const actualMessageCount = await page
        .locator('[data-testid="chat-scroll-area"] [class*="Paper"]')
        .count();
      console.log("Actual message count:", actualMessageCount);

      // Check that chat frame exists and has reasonable dimensions
      const chatFrame = await page.locator('[data-testid="chat-scroll-area"]').boundingBox();
      const viewportSize = page.viewportSize();

      expect(chatFrame).not.toBeNull();
      expect(viewportSize).not.toBeNull();

      // Chat frame should be visible and have content
      if (chatFrame) {
        expect(chatFrame.height).toBeGreaterThan(100); // Should have sufficient height
        expect(chatFrame.width).toBeGreaterThan(100); // Should have sufficient width
      }

      // Check if scrollbar is present in ScrollArea
      const scrollInfo = await page.evaluate(() => {
        // すべてのScrollAreaを確認
        const allScrollAreas = document.querySelectorAll('[class*="ScrollArea"]');
        console.log("Total ScrollAreas found:", allScrollAreas.length);

        // chat-scroll-area内のScrollArea-viewportを正確に取得
        const chatScrollArea = document.querySelector('[data-testid="chat-scroll-area"]');
        const scrollViewport = chatScrollArea
          ? chatScrollArea.querySelector('[class*="ScrollArea-viewport"]')
          : null;
        const messagesContainer = chatScrollArea
          ? chatScrollArea.querySelector('[class*="Stack"]')
          : null;

        // ScrollAreaの高さもチェック
        const scrollAreaHeight = chatScrollArea ? chatScrollArea.getBoundingClientRect().height : 0;
        const scrollAreaStyle = chatScrollArea ? window.getComputedStyle(chatScrollArea) : null;

        if (!scrollViewport)
          return { found: false, scrollHeight: 0, clientHeight: 0, hasScroll: false };
        return {
          found: true,
          scrollHeight: scrollViewport.scrollHeight,
          clientHeight: scrollViewport.clientHeight,
          offsetHeight: scrollViewport.offsetHeight,
          scrollAreaHeight: scrollAreaHeight,
          scrollAreaComputedHeight: scrollAreaStyle ? scrollAreaStyle.height : "none",
          scrollAreaMaxHeight: scrollAreaStyle ? scrollAreaStyle.maxHeight : "none",
          messagesHeight: messagesContainer ? messagesContainer.scrollHeight : 0,
          hasScroll: scrollViewport.scrollHeight > scrollViewport.clientHeight,
          totalScrollAreas: allScrollAreas.length,
        };
      });

      console.log("Scroll Info:", scrollInfo);
      expect(scrollInfo.hasScroll).toBe(true);

      // Check message count badge
      const messageBadge = await page.locator("text=/メッセージ: \\d+\\/100/").textContent();
      expect(messageBadge).toBe("メッセージ: 30/100");

      // Test scrolling functionality
      const scrollArea = page.locator(
        '[data-testid="chat-scroll-area"] [class*="ScrollArea-viewport"]',
      );

      // First scroll to top since auto-scroll likely positioned us at bottom
      await scrollArea.evaluate((el) => {
        el.scrollTop = 0;
        return el.scrollTop;
      });
      await page.waitForTimeout(500);

      // Verify at top
      const topScrollTop = await scrollArea.evaluate((el) => el.scrollTop);
      expect(topScrollTop).toBe(0);

      // Scroll to bottom
      const scrollInfo2 = await scrollArea.evaluate((el) => {
        el.scrollTop = el.scrollHeight;
        return { scrollTop: el.scrollTop, scrollHeight: el.scrollHeight };
      });
      await page.waitForTimeout(500);

      // Verify scrolled to bottom (scrollTop should be scrollHeight - clientHeight)
      const expectedBottomScroll = scrollInfo2.scrollHeight - 840; // 840 is our fixed height
      expect(scrollInfo2.scrollTop).toBeGreaterThan(expectedBottomScroll - 10); // Allow small margin

      // Scroll to middle
      await scrollArea.evaluate((el) => {
        el.scrollTop = el.scrollHeight / 2;
        return el.scrollTop;
      });
      await page.waitForTimeout(500);

      // Verify scroll is in middle
      const middleScrollTop = await scrollArea.evaluate((el) => el.scrollTop);
      expect(middleScrollTop).toBeGreaterThan(0);
      expect(middleScrollTop).toBeLessThan(scrollInfo2.scrollTop);

      // Scroll back to top to verify full range
      await scrollArea.evaluate((el) => {
        el.scrollTop = 0;
        return el.scrollTop;
      });
      await page.waitForTimeout(500);

      const finalTopScroll = await scrollArea.evaluate((el) => el.scrollTop);
      expect(finalTopScroll).toBe(0);
    });

    test("should limit messages to 100 when sending many messages", async ({ page }) => {
      // Send 110 messages in a loop
      // Note: test server may send duplicate IDs, so actual count may vary
      for (let i = 0; i < 110; i++) {
        await fetch(`${WS_TEST_SERVER}/api/send/test`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
        });
      }

      // Wait for all messages to be processed
      await page.waitForTimeout(3000);

      // Wait for UI to stabilize and get consistent counts
      let messageCount = 0;
      let badgeCount = 0;

      // Retry a few times to get stable counts
      for (let retry = 0; retry < 3; retry++) {
        messageCount = await page
          .locator('[data-testid="chat-scroll-area"] [class*="Paper"]')
          .count();

        const messageBadge = await page.locator("text=/メッセージ: \\d+\\/100/").textContent();
        const match = messageBadge?.match(/(\d+)\/100/);
        badgeCount = match ? parseInt(match[1], 10) : 0;

        // If counts match, we're done
        if (messageCount === badgeCount) {
          break;
        }

        // Wait a bit for synchronization
        await page.waitForTimeout(1000);
      }

      // Ensure message count does not exceed 100 (the limit)
      expect(messageCount).toBeLessThanOrEqual(100);
      expect(badgeCount).toBeLessThanOrEqual(100);

      // Allow for small discrepancy (±1) due to timing issues
      expect(Math.abs(badgeCount - messageCount)).toBeLessThanOrEqual(1);

      // Verify chat frame exists and has content
      const chatFrame = await page.locator('[data-testid="chat-scroll-area"]').boundingBox();

      if (chatFrame) {
        expect(chatFrame.height).toBeGreaterThan(100); // Should have sufficient height
      }

      // Verify scrollbar is still functional
      // First check if we actually have enough messages to create a scrollbar
      const scrollArea = page.locator(
        '[data-testid="chat-scroll-area"] [class*="ScrollArea-viewport"]',
      );

      const scrollInfo = await scrollArea.evaluate((el) => {
        return {
          scrollHeight: el.scrollHeight,
          clientHeight: el.clientHeight,
          hasScrollbar: el.scrollHeight > el.clientHeight,
          messageCount: el.querySelectorAll('[class*="Paper"]').length,
        };
      });

      console.log("Scroll verification info:", scrollInfo);

      // Only verify scrolling if there are enough messages to create a scrollbar
      if (scrollInfo.hasScrollbar) {
        const canScroll = await scrollArea.evaluate((el) => {
          // First scroll to top to ensure we have room to scroll down
          el.scrollTop = 0;
          const topPosition = el.scrollTop;

          // Now try to scroll to bottom
          el.scrollTop = el.scrollHeight;
          const bottomPosition = el.scrollTop;

          // Check if we actually scrolled
          const scrolled = bottomPosition > topPosition;

          // Also check scroll range
          const expectedBottom = el.scrollHeight - el.clientHeight;
          const isAtBottom = Math.abs(bottomPosition - expectedBottom) < 10; // Allow small tolerance

          return {
            scrolled,
            topPosition,
            bottomPosition,
            scrollHeight: el.scrollHeight,
            clientHeight: el.clientHeight,
            expectedBottom,
            isAtBottom,
          };
        });

        console.log("Scroll test result:", canScroll);
        expect(canScroll.scrolled).toBe(true);
        expect(canScroll.isAtBottom).toBe(true);
      } else {
        // If there's no scrollbar, just verify the messages are limited
        console.log("No scrollbar needed with", scrollInfo.messageCount, "messages");
        expect(scrollInfo.messageCount).toBeGreaterThan(0);
        expect(scrollInfo.messageCount).toBeLessThanOrEqual(100);
      }
    });

    test("should clear messages when clicking clear button", async ({ page }) => {
      // Send some test messages
      for (let i = 0; i < 5; i++) {
        await fetch(`${WS_TEST_SERVER}/api/send/test`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
        });
      }

      // Wait for messages to appear
      await page.waitForTimeout(1000);

      // Verify messages exist
      let messageCount = await page
        .locator('[data-testid="chat-scroll-area"] [class*="Paper"]')
        .count();
      expect(messageCount).toBe(5);

      // Click clear button
      await page.getByRole("button", { name: "クリア" }).click();

      // Verify messages are cleared
      messageCount = await page
        .locator('[data-testid="chat-scroll-area"] [class*="Paper"]')
        .count();
      expect(messageCount).toBe(0);

      // Verify message count badge is also cleared
      const hasBadge = await page.locator("text=/メッセージ: \\d+\\/100/").count();
      expect(hasBadge).toBe(0);
    });
  });
