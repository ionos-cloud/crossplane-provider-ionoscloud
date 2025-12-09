# ====================================================================================
# Setup Project

include ../../../build/makelib/common.mk

# ====================================================================================
#  Options
include ../../../build/makelib/imagelight.mk

# ====================================================================================
# Targets

img.publish:
	@$(INFO) Skipping image publish for $(IMAGE)
	@echo Publish is deferred to xpkg machinery
	@$(OK) Image publish skipped for $(IMAGE)

img.build:
	@$(INFO) docker build $(IMAGE)
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORM) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)
	@$(OK) docker build $(IMAGE)

img.promote:
	@$(INFO) Skipping image promotion from $(FROM_IMAGE) to $(TO_IMAGE)
	@echo Promote is deferred to xpkg machinery
	@$(OK) Image promotion skipped for $(FROM_IMAGE) to $(TO_IMAGE)
