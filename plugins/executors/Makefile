


TOPTARGETS := build clean

PLUGINS = skeleton

$(TOPTARGETS): $(PLUGINS)
$(PLUGINS):
	@echo "Building $@:"
	@$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: $(TOPTARGETS) $(PLUGINS)
