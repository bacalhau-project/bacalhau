package hook

import (
	"github.com/spf13/cobra"
)

// A Cobra run hook that cannot return an error.
type runHook = func(*cobra.Command, []string)

// A Cobra run hook that may return an error.
type runHookE = func(*cobra.Command, []string) error

// Run all of the passed hooks in order, stopping on and returning the first
// error.
func Chain(hooks ...runHookE) runHookE {
	return func(c *cobra.Command, s []string) error {
		for _, hook := range hooks {
			if err := hook(c, s); err != nil {
				return err
			}
		}
		return nil
	}
}

// Adapt turns a run hook that does not return an error into one that does.
func Adapt(hook runHook) runHookE {
	return func(c *cobra.Command, s []string) error {
		hook(c, s)
		return nil
	}
}

// Because cobra doesn't do PersistentPreRun{,E} chaining yet
// (https://github.com/spf13/cobra/issues/252), this function walks upwards and
// finds any parent hook that should be run if the hooks were properly
// persistent. The complexity here is that if we are running on a child command
// we will first find the parent hook, and need to skip that otherwise we'll end
// up in an infinite loop.
func AfterParentHook(hook runHookE, getHook func(*cobra.Command) runHookE) runHookE {
	return func(c *cobra.Command, s []string) error {
		seenThis := false
		for cmd := c; cmd != nil; cmd = cmd.Parent() {
			possibleParentHook := getHook(cmd)
			if possibleParentHook != nil && seenThis {
				err := possibleParentHook(c, s)
				if err != nil {
					return err
				}
				break
			} else if possibleParentHook != nil && !seenThis {
				seenThis = true
			}
		}
		return hook(c, s)
	}
}

func AfterParentPreRunHook(hook runHookE) runHookE {
	return AfterParentHook(hook, func(c *cobra.Command) runHookE { return c.PersistentPreRunE })
}

func AfterParentPostRunHook(hook runHookE) runHookE {
	return AfterParentHook(hook, func(c *cobra.Command) runHookE { return c.PersistentPostRunE })
}

// ClientPreRunHooks is the set of pre-run hooks that all client commands
// should have applied.
var ClientPreRunHooks runHookE = Chain(
	Adapt(ApplyPorcelainLogLevel),
	// TODO(forrest) [fixme/are-you-fucking-kidding-me]: need to un-wang this from here
	// gut check idea is to perform this in a single place at the root level
	// Adapt(StartUpdateCheck),
)

// RemoteCmdPreRunHooks is the set of pre-run hooks that all commands that
// communicate with remote servers should have applied.
var RemoteCmdPreRunHooks runHookE = Chain(
	ClientPreRunHooks,
)

// ClientPostRunHooks is the set of post-run hooks that all client commands
// should have applied.
var ClientPostRunHooks runHookE = Chain(
	Adapt(PrintUpdateCheck),
)

// RemoteCmdPostRunHooks is the set of post-run hooks that all commands that
// communicate with remote servers should have applied.
var RemoteCmdPostRunHooks runHookE = ClientPostRunHooks
