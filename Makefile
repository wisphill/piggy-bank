APP_NAME := piggy-bank
TARGET_DIR := target
BINARY := $(TARGET_DIR)/$(APP_NAME)
APP_BUNDLE := PiggyBank.app
APP_BUNDLE_MACOS := $(APP_BUNDLE)/Contents/MacOS

.PHONY: run build clean app install dmg pkg

run:
	CGO_ENABLED=1 go run .

build:
	mkdir -p $(TARGET_DIR)
	CGO_ENABLED=1 go build -o $(BINARY) .

app: build
	mkdir -p $(APP_BUNDLE_MACOS)
	mkdir -p $(APP_BUNDLE)/Contents
	cp $(BINARY) $(APP_BUNDLE_MACOS)/$(APP_NAME)
	cp -r bundle/Contents/* $(APP_BUNDLE)/Contents
	@echo "✓ PiggyBank built successfully"

install: app
	mkdir -p ~/Applications
	cp -r $(APP_BUNDLE) ~/Applications/$(APP_BUNDLE)
	@echo "✓ PiggyBank installed to ~/Applications"

start: build
	./$(BINARY)

clean:
	rm -f $(APP_NAME)
	go clean

# Create DMG installer (standard macOS distribution)
dmg: app
	@echo "📦 Creating DMG installer..."
	mkdir -p $(TARGET_DIR)
	rm -f $(TARGET_DIR)/PiggyBank.dmg
	mkdir -p /tmp/PiggyBank-dmg
	cp -r $(APP_BUNDLE) /tmp/PiggyBank-dmg/
	ln -s /Applications /tmp/PiggyBank-dmg/Applications 2>/dev/null || true
	hdiutil create -volname "PiggyBank" -srcfolder /tmp/PiggyBank-dmg -ov -format UDZO $(TARGET_DIR)/PiggyBank.dmg
	rm -rf /tmp/PiggyBank-dmg
	@echo "✓ DMG installer created: $(TARGET_DIR)/PiggyBank.dmg"

# Create PKG installer (traditional "next next" installer with UI)
pkg: app
	@echo "📦 Creating PKG installer..."
	mkdir -p $(TARGET_DIR)/PiggyBank-pkg/Applications
	cp -r $(APP_BUNDLE) $(TARGET_DIR)/PiggyBank-pkg/Applications/
	pkgbuild --root $(TARGET_DIR)/PiggyBank-pkg \
		--identifier com.wisphill.PiggyBank \
		--version 1.0 \
		--install-location / \
		$(TARGET_DIR)/PiggyBank-Installer.pkg
	rm -rf $(TARGET_DIR)/PiggyBank-pkg
	@echo "✓ PKG installer created: $(TARGET_DIR)/PiggyBank-Installer.pkg"
	@echo "👉 Double-click to install with UI!"
