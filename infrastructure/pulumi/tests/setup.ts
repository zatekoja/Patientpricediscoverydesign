// Test setup file - runs before all tests
import * as pulumi from '@pulumi/pulumi';

// Set Pulumi to dry-run mode for unit tests
pulumi.runtime.setMocks({
  newResource: (args: pulumi.runtime.MockResourceArgs): { id: string; state: any } => {
    return {
      id: `${args.name}-id`,
      state: args.inputs,
    };
  },
  call: (args: pulumi.runtime.MockCallArgs) => {
    return args.inputs;
  },
});
