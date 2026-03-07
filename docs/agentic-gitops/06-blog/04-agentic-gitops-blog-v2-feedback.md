I like the idea of the inner loop and outer loop.
But I think many people will not understand what this really means.
The terms are also used in other contexts. For example, some people use inner loop for Kubernetes local development and outer loop for Docker or cluster workflows. So maybe this needs clearer explanation.

I also like that every change must now be checked. It must be clear if the change was authorized and how it maps to the real world. This governance idea is strong and important .

Maybe I understand it like this to get a full overview:

* cub-track is integrated into the repo and tracks commits, PRs, and changes.
* cub-scout watches the live cluster, because ConfigHub defines the desired state for the manifest.
* Git is mainly for collaboration, like it was planned before.

So almost the full scope is covered.

If cub-scout finds an error in the live cluster, it reports it and maybe suggests how to fix it. A human can then apply the change in Git. Tests run, there is a merge, and then GitOps and the reconciler apply it again.

The same the other way around: if cub-track finds a problem, like a security issue or inconsistency, does it create a PR with a possible fix?

Does this mean cub-scout must should in the cluster (in the future), detect problems, and maybe automatically create a branch and PR, similar to the Kargo principle?
Then cub-track reviews and validates it, because it knows the Git history, issues, and PRs. A human is still in the loop and must approve.
But of course, cub-track can also opperate on their own.

So it becomes something like bidirectional GitOps with branches, even if branches are normally seen as an anti-pattern. They are just a technical tool, like in Kargo, but supported by agents with AI capabilities .

One thing I think is missing is the dynamic nature of Kubernetes.
There are event-driven systems, autoscaling like HPA, admission controllers that mutate resources at runtime, and native controllers that change things automatically. These already create problems with GitOps today. Often we just ignore some fields, like replicas with HPA, to keep GitOps working, but this creates gaps.

Also, when a problem happens, we as humans check multiple data sources:

1. We look into the cluster: Argo apps, events, deployments.
2. We check Git history and PRs. Who made the last change?
3. We check logs, metrics, traces. Maybe the app normally uses 2 CPUs, but sometimes peaks to 4 CPUs at night. If a new change sets the limit to 2.5 CPUs, this can create problems.

If the system does not understand this historical and runtime context, we can create an endless loop.
Fix one thing, break another thing, fix again, and so on. This could be a weakness in the current idea.
we saw in similar worklflows, it was funny, because that are just test systems and no one expect real ouput 2 years ago.

I am also not fully sure how the interaction between all components should work in practice. It feels very dynamic and could result in thousands of PRs.

Today we already have tools like Renovate Bot. They check dependencies, CVEs, library updates, and create PRs based on rules.
They create already a lot of PRs and increasing the cognitive load for the reviewers.
After approval, tests run, it gets merged, and the reconciler applies it to the cluster. This is simple and rule-based.

Two years ago, we tried something similar. We built a system with different agents:

* Entry Agent
* Git Agent (similar to cub-track)
* Observability Agents for cluster, logs, metrics, traces
* Helm Agent
* GitOps Agent for Argo CD or Flux
* Kubernetes Agent
* Policy Agent
* PR Agent

Depending on the use case, there was a different flow between the agents.
In the end, a PR was created with a suggested fix. Tests and validations ran. A human could review the fix, see the sources and agents involved, and then approve it.

A friend from GitHub was visiting me and we talked about how the new development workflow with ai will looks like.
At GitHub the see it more like now you are becoming more than a reviewer with you seniors experience like you review junior dev code
in the past, but now you are reviewing it from AI software defined apps or infra.
The challenge is how to bring it secure into production by allowing changes be easyy revertable and

I like the idea and I think of course think about multiple topics here like the event-driven part, how observbility fits in, what is bidirectional GitOps and how it fits into the GitOps reconciler pattern (I like the way from Kargo and Codefresh, you also showing examples from them in your Keynote).

I also think we need to definied the most important parts
of our daily works like Git, GitOps, Obserbility, Security,
Signing, Attestation, etc. and see how the puzzle fits together with the agentic GitOps approach.


