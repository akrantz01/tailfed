SRCS := $(shell find . -type f -name "*.tf")
TERRAFORM := tofu

.PHONY: apply destroy fmt help init plan

help:
	$(info help    - display this message)
	$(info init    - initialize the configuration)
	$(info fmt     - format the configuration)
	$(info plan    - perform a speculative plan)
	$(info apply   - create/update the resources)
	$(info destroy - destory all the resources)

init: .tf-init

fmt: $(SRCS)
	$(TERRAFORM) fmt

plan: $(SRCS)
	$(TERRAFORM) plan

apply: $(SRCS)
	$(TERRAFORM) apply

destroy:
	$(TERRAFORM) destroy

.tf-init: main.tf .terraform.lock.hcl
	$(TERRAFORM) init
	@touch .tf-init
