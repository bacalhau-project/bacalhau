TOPTARGETS := build clean

SUBDIRS := executors

$(TOPTARGETS): $(SUBDIRS)
$(SUBDIRS):
	@echo "$@:"
	@$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: $(TOPTARGETS) $(SUBDIRS)