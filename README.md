# visibility-testing

This repo contains a mirror of the instructions of how to deploy incident-io/core, our Go monolith.

We use this to power agents that monitor our deployment pipeline. It's essential we keep the documentation below in-sync with that repo so that our deploys work successfully, or we might risk an incident.

In general, deploys should go to environments in this order:

- alpha: is where we deploy first
- beta: is where we then load-test if alpha booted correctly
- theta: this is what we refer to as production

You can find deployment docs in the incident-io/core repo base that show how this works.
