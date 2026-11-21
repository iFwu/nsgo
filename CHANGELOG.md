# Changelog

## [Unreleased]

- Free an overlay, a `Tooltip`'s included, once the view it belongs to is removed, instead of its host keeping it, and through it the whole subtree it was in, for the life of the window.
- Free a removed view that was laid out on its own as a relayout boundary, instead of the queue of boundaries keeping it, and through it the whole subtree it was in, until a later queue as long reuses its slot.
- Draw a multiline `Input`'s lines `Style.LineHeight` apart on macOS, its placeholder's included, with their glyphs where a `Text` of its style puts them, as the engine lays the input out, instead of at the font's own height.
- Land a `List`'s scroll to a row, or to its end, where it asks when rows were inserted, removed or measured above the viewport in the same update, instead of shifted by the scroll that would have kept the visible rows in place.
- Pull a scroll view's offset back to its end in `nsgotest` when its content shrinks or its viewport grows past it, and report where it went, as macOS does, instead of leaving it past the end.
- Free the views a view's children, a `Dynamic`, an `Each` or a text's attachments drop, and those last told their layout, instead of the lists that held them keeping them past their new end, and through each the whole subtree it was in, until the list grows back that long.
- Free an absolutely positioned or fixed view its containing block no longer has, instead of the containing block's list of the boxes it positions keeping it past its new end, and through it the whole subtree it was in, until the containing block positions as many again.
- Free a removed `EdgeBar` view and a removed scroll view a bar was over, instead of the lists of bars and of the scroll views they reach into keeping them past their new end, and through each the whole subtree it was in, while another bar stays.
- Record in `nsgotest` the scrolls an app asks a scroll view for, to a point or by a distance and whether animated, in `View.Scrolls`, as `WebCommands` and `Seeks` record what a web view and a video were asked.
- Free a removed `List`, instead of the tree's list of the lists it updates keeping it past its new end, and through it the whole subtree it was in, while another list stays.
- Let a `List`'s row whose views have a closed tooltip go once it scrolls away, instead of holding it as a row with an open overlay is held, which kept every such row mounted, and in memory, for as long as the list stayed.
- Keep a multiline `Input`'s caret, its selection and the rects an input method places its candidates by to the font's ascent and descent on macOS when `Style.LineHeight` is taller than the font, instead of reaching up through the extra height, with the glyphs where they were.
- Drag a scroll view in `nsgotest` by the distance `Scroll` asks for from wherever the content is, as a finger does, instead of on to the offset it started for, which undid the scroll by a `List` keeping its visible rows in place as rows above them turned out shorter than estimated, and moved those rows instead.
- Keep a keyed `List` row in its cell, with its measured size and, as the first visible row, its place on screen, when one update changes rows on both sides of it, instead of handing its cell to another row and building it again, which dropped what it held, such as a field's focus and its input method's composition.
- Add `Style.GlassInteractive`, and `glassInteractive` in TSX, which has a view's Liquid Glass answer the presses the view takes as the system's own controls of glass do on macOS 27, swelling under the pointer and springing back.
- Report the original GIF in an input's `OnPaste` on macOS, even when the pasteboard also offers a still image.

## [0.2.0] - 2026-11-19

- Add HarmonyOS as a preview target, whose ArkTS host runs the engine in ArkWeb and draws it with ArkUI views, in `nsgo build`, `nsgo run` and `nsgo test -target harmonyos` (#79 by @togotop).
- Add in-app purchases on macOS, iOS and tvOS through StoreKit 2: the store's products and prices, purchases answering cancelled and pending apart from failed with the signed transaction, updates made outside the app, finishing, current entitlements and restoring, in Go, TSX and `nsgotest`, when `nsgo.yaml` sets `apple.purchases` (#353 by @houko).
- Add `desktop.SetWindowWorkspaces` and `WindowOptions.Workspaces`, which keep a window on every Space and over other apps' full screen spaces on macOS, at any level but the desktop's, which stays behind them, and on every virtual desktop on X11, with `desktop.CanSetWindowWorkspaces`, which says whether it takes effect (#572 by @houko).
- Add transparent desktop windows without a frame, shadow or taskbar button, as `WindowOptions.Transparent`, `NoShadow` and `SkipTaskbar`, the main window's `nsgo.WindowTransparent`, `WindowNoShadow` and `SkipTaskbar`, and `desktop.SetWindowTransparent`, `SetWindowShadow` and `SetSkipTaskbar`, on macOS, Windows and Linux, with `desktop.CanMakeWindowsTransparent` and `CanSkipTaskbar` saying where they take effect (#567 by @houko).
- Add `PanelOptions.Transparent`, a panel with no background, frame or shadow of its own whose content draws its corners and shadow, and `PanelOptions.FitContent`, a panel sized to its content before it first shows and as it changes, on macOS, Windows and Linux, with `desktop.Transparent` saying whether a transparent panel or window draws nothing of its own now (#315 by @houko).
- Add `SidebarShell.Fixed`, a sidebar the user cannot resize.
- Add `desktop.SetWindowPointerEvents`, with `WindowOptions.PointerEvents` and `PanelOptions.PointerEvents`, which lets the pointer through a window to the windows below it everywhere or wherever it draws nothing, on macOS, Windows and Linux (#568 by @houko).
- Add `desktop.FrontmostApp`, the application in front on the desktop with its ID, name, process ID and full screen, tracked as other apps come to the front, on macOS, Windows and X11, and `desktop.CanReadFrontmostApp`, which says where it is followed; both are held to `core.AccessFrontmostApp` and set in tests through `nsgotest.Environment` (#570 by @houko).
- Add `ImageSource.Pixels` and `Version`, which show an `image.RGBA` the app draws in an `Image` or a `Canvas` with its alpha, no load events and no fade, swapped in place without encoding on Apple platforms, Windows and Linux (#107 by @houko).
- Add `Scope.SetWebCookies`, which writes cookies, HttpOnly ones included, to the persistent store the app's web views share on Apple platforms, Windows, Linux and Android, so they can carry the sign-in of the user's own browser (#387 by @houko).
- Add `desktop.CanSetWindowLevel` and `desktop.CanPlaceWindows`, and keep a floating window above the others on Linux under X11, so a Wayland app can tell that its window is neither stacked nor placed (#573 by @houko).
- Add `WebView.PinchZoom`, `Ref.Load` and the platform's `Domain` and `Code` on `WebError`; on Windows, Ctrl with the wheel or with + and - zooms a web view only with `PinchZoom` (#233 by @houko).
- Add `core.ScrollEvent.Stale`, TSX's `stale`, which marks a report from before the platform applied a scroll from code, and `terminal.View.HideIndicator`, which hides its scroll bar (#692 by @houko).
- Add `chat.Composer.AutoFocus`, which focuses the editor as the composer mounts.
- Add `chat.QuotaRow.Pace`, which marks on a limit's bar in `chat.Quota` and `chat.QuotaSummary` what would remain had it been used evenly, so a fill short of the mark runs out before it resets.
- Add `ContentWidth` to `chat.Conversation` and `chat.FloatingConversation`, which centers the transcript and the floating composer in at most that width, and fill the conversation without it, replacing the fixed 720 pt `chat.ContentWidth` (#719).
- Report `Video.OnError` instead of an opaque picture for a local video whose alpha the platform cannot keep: WebM with alpha on macOS and iOS, any clip with alpha on Windows until Media Foundation's alpha output is checked, and on Linux HEVC with alpha, or VP8 and VP9 with alpha without GStreamer's codecalpha plug-in; add `MediaError.Reason`, `MediaErrorAlphaUnsupported` for these, so an app can fall back to its own decoder without matching the message (#569 by @houko).
- Give a `Select`'s and a `DatePicker`'s field the height, padding and type their `Style` sets, as an `Input`'s, instead of only the view around it, so a small one lines up with a `SizeSmall` button (#721 by @Session500).
- Stop reporting a web view load that was stopped, replaced or turned into a download to `OnError` on Apple platforms and Windows (#233 by @houko).
- Keep a view's own outer shadows and outline outside its `Overflow: Hidden` clip on Apple for views, images and macOS scroll views, as CSS does (#238 by @houko).
- Lay out and draw a Windows window at each size as a drag resizes it, as macOS does, instead of stretching the last frame to the window.
- Show `chat.FloatingConversation`'s bottom edge blur only while a composer is shown, so a transcript without one ends sharp.
- Keep a `base/terminal` view following output that comes faster than frames instead of leaving the end, and follow it again once scrolled back to the end (#686).
- Keep a multiline `Input`'s scroll bar along its edges on macOS and iOS, its padding insetting the text, which scrolls under it (#701).
- Keep a scroll view still as a text selection starts in it on macOS, rather than scrolling it toward where the last drag ended or the window's corner.
- Open the output directory with `nsgo build -open` when nsgo leads its terminal's session, as in an app's terminal without a shell, instead of losing the opener as nsgo exits (#672).
- Show a bundled macOS app's Dock badge without the notification permission until the app uses notifications (#190 by @houko).
- Accept a bare OS name in `nsgo check -target`, `nsgo build -target` and `nsgo.yaml`'s `build.targets`, and an arch-qualified name in `nsgo test -target`, so the commands take each other's target spellings (#593 by @dhruv1220).
- Keep an engine fault's `CheckCommits` re-panic ahead of its exit notification, so fault-isolation tests cannot mistake the notification for a swallowed panic (#703).
- Stop canvases and shaders on Apple platforms and the web from drawing an earlier image for an image's inline `Data` once the earlier buffer was freed and its address reused, by keeping a cached image's bytes alive (#715, #737, #751).
- Localize the separator between Android TalkBack's state parts with the app's language, including Chinese punctuation (#704).

## [0.1.0] - 2026-11-16

NSGo 0.1.0 is the first release of a cross-platform native UI framework in Go, porting NSH's design: write an app once, in Go or TSX, and it runs on phones, tablets, TVs, watches, desktops and the web in each platform's own views. Its API is not stable yet.

- **Every platform from one Mac.** Build for macOS, iOS with App Clips, tvOS, watchOS, Windows, Linux, Android and the web with only Go and Xcode, and for WeChat, QQ, Alipay and Douyin mini programs experimentally.
- **Native views.** Text, input methods, scrolling, menus and accessibility are each platform's own, and what one platform has, such as Liquid Glass, the others imitate.
- **Fine-grained updates.** Signals update only what reads them, through flexbox and grid layout, CSS-like styles, Tailwind classes, themes and animations on the platform's frame clock.
- **Components.** `base` offers unstyled components from buttons to a million-row `DataTable`, charts, flow diagrams, a code editor, a rich-text editor and a terminal, and `ui` gives them NSGo's look.
- **Platform services.** Use navigation, notifications, permissions, pickers, sharing, haptics, widgets and Live Activities, and on desktops windows, panels, menus, tray icons and global shortcuts.
- **TSX apps.** Write apps in TSX as NSH does, with hot updates that keep state, and ship them as signed, sandboxed packages a Go app runs.
- **Localized and embeddable.** Translate with YAML catalogs, CLDR plurals and right-to-left layout, and embed NSGo in iOS, Android, Go desktop and web apps.
- **The `nsgo` tool.** Create apps, run them with hot reload and DevTools, test them with `nsgotest`, and package them for app stores, Flathub and direct downloads with updates.
- **Small and fast.** A counter app packs to about 8 MB on the desktops and starts faster than NSH, MyGo and GPUI in NSGo's benchmarks.

Thanks to everyone who shaped this release with code, reports and ideas: @houko, @daodao97, @Heyana, @ayuayue, @henry-wm, @Jungley8, @togotop, @westhiker, @Laotree, @OfficialDiLi, @happyhaha1, @roamlog, @yangjuncode, @pdenapo and @monkeyWie.
