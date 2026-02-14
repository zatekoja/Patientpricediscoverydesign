import * as pulumi from '@pulumi/pulumi';
import * as aws from '@pulumi/aws';
import * as random from '@pulumi/random';
import { getResourceTags } from '../tagging';

export interface SecretsConfig {
  environment: string;
  databasePassword?: pulumi.Output<string>;
  redisAuthToken?: pulumi.Output<string>;
  blnkRedisAuthToken?: pulumi.Output<string>;
  jwtSecret?: pulumi.Output<string>;
  apiKeys?: Record<string, string>;
}

export interface SecretsOutputs {
  masterSecretArn: pulumi.Output<string>;
  masterSecretName: pulumi.Output<string>;
  databasePasswordArn: pulumi.Output<string>;
  redisAuthTokenArn: pulumi.Output<string>;
  blnkRedisAuthTokenArn: pulumi.Output<string>;
  jwtSecretArn?: pulumi.Output<string>;
}

/**
 * Create master secret with all credentials
 */
export function createMasterSecret(config: SecretsConfig): aws.secretsmanager.Secret {
  return new aws.secretsmanager.Secret(`ohi-${config.environment}-master`, {
    name: `ohi-${config.environment}-master`,
    description: `Master secret for OHI ${config.environment} containing all credentials`,
    recoveryWindowInDays: config.environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(config.environment, 'master-secret') },
  });
}

/**
 * Create master secret version with all credentials
 */
export function createMasterSecretVersion(
  secret: aws.secretsmanager.Secret,
  config: SecretsConfig
): aws.secretsmanager.SecretVersion {
  const secretData = pulumi
    .all([
      config.databasePassword || pulumi.output(''),
      config.redisAuthToken || pulumi.output(''),
      config.blnkRedisAuthToken || pulumi.output(''),
      config.jwtSecret || pulumi.output(''),
    ])
    .apply(([dbPass, redisToken, blnkRedisToken, jwt]) => {
      const data: Record<string, string> = {
        DATABASE_PASSWORD: dbPass || '',
        REDIS_AUTH_TOKEN: redisToken || '',
        BLNK_REDIS_AUTH_TOKEN: blnkRedisToken || '',
      };

      if (jwt) {
        data.JWT_SECRET = jwt;
      }

      if (config.apiKeys) {
        Object.entries(config.apiKeys).forEach(([key, value]) => {
          data[`API_KEY_${key.toUpperCase()}`] = value;
        });
      }

      return JSON.stringify(data);
    });

  return new aws.secretsmanager.SecretVersion(`ohi-${config.environment}-master-version`, {
    secretId: secret.id,
    secretString: secretData,
  });
}

/**
 * Create individual secret for database password
 */
export function createDatabasePasswordSecret(
  environment: string,
  password: pulumi.Output<string>
): aws.secretsmanager.Secret {
  const secret = new aws.secretsmanager.Secret(`ohi-${environment}-db-password`, {
    name: `ohi-${environment}-db-password`,
    description: `Database password for OHI ${environment}`,
    recoveryWindowInDays: environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(environment, 'db-password-secret') },
  });

  new aws.secretsmanager.SecretVersion(`ohi-${environment}-db-password-version`, {
    secretId: secret.id,
    secretString: password,
  });

  return secret;
}

/**
 * Create individual secret for Redis auth token
 */
export function createRedisAuthTokenSecret(
  environment: string,
  token: pulumi.Output<string>,
  isBlnk: boolean = false
): aws.secretsmanager.Secret {
  const name = isBlnk ? `ohi-${environment}-blnk-redis-token` : `ohi-${environment}-redis-token`;

  const secret = new aws.secretsmanager.Secret(name, {
    name: name,
    description: `Redis auth token for OHI ${environment}${isBlnk ? ' (Blnk)' : ''}`,
    recoveryWindowInDays: environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(environment, `redis-token-secret${isBlnk ? '-blnk' : ''}`) },
  });

  new aws.secretsmanager.SecretVersion(`${name}-version`, {
    secretId: secret.id,
    secretString: token,
  });

  return secret;
}

/**
 * Generate JWT secret
 */
export function generateJwtSecret(environment: string): random.RandomPassword {
  return new random.RandomPassword(`ohi-${environment}-jwt-secret`, {
    length: 64,
    special: true,
    overrideSpecial: '!@#$%^&*()_+-=[]{}|;:,.<>?',
  });
}

/**
 * Create JWT secret in Secrets Manager
 */
export function createJwtSecret(
  environment: string,
  jwtSecret: pulumi.Output<string>
): aws.secretsmanager.Secret {
  const secret = new aws.secretsmanager.Secret(`ohi-${environment}-jwt-secret`, {
    name: `ohi-${environment}-jwt-secret`,
    description: `JWT signing secret for OHI ${environment}`,
    recoveryWindowInDays: environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(environment, 'jwt-secret') },
  });

  new aws.secretsmanager.SecretVersion(`ohi-${environment}-jwt-secret-version`, {
    secretId: secret.id,
    secretString: jwtSecret,
  });

  return secret;
}

/**
 * Create IAM policy for ECS task to access secrets
 */
export function createSecretsAccessPolicy(
  environment: string,
  secretArns: pulumi.Output<string>[]
): aws.iam.Policy {
  const policyDocument = pulumi.all(secretArns).apply((arns) =>
    JSON.stringify({
      Version: '2012-10-17',
      Statement: [
        {
          Effect: 'Allow',
          Action: ['secretsmanager:GetSecretValue', 'secretsmanager:DescribeSecret'],
          Resource: arns,
        },
        {
          Effect: 'Allow',
          Action: ['kms:Decrypt'],
          Resource: '*', // KMS key for Secrets Manager encryption
          Condition: {
            StringEquals: {
              'kms:ViaService': `secretsmanager.eu-west-1.amazonaws.com`,
            },
          },
        },
      ],
    })
  );

  return new aws.iam.Policy(`ohi-${environment}-secrets-access`, {
    name: `ohi-${environment}-secrets-access`,
    description: `Policy for ECS tasks to access secrets in ${environment}`,
    policy: policyDocument,
    tags: { ...getResourceTags(environment, 'secrets-access-policy') },
  });
}

/**
 * Attach secrets policy to ECS task execution role
 */
export function attachSecretsPolicyToRole(
  environment: string,
  roleName: pulumi.Output<string>,
  policy: aws.iam.Policy
): aws.iam.RolePolicyAttachment {
  return new aws.iam.RolePolicyAttachment(`ohi-${environment}-secrets-policy-attachment`, {
    role: roleName,
    policyArn: policy.arn,
  });
}

/**
 * Create complete secrets infrastructure
 */
export function createSecretsInfrastructure(config: SecretsConfig): SecretsOutputs {
  // Create individual secrets
  const dbPasswordSecret = createDatabasePasswordSecret(
    config.environment,
    config.databasePassword || pulumi.output('')
  );

  const redisTokenSecret = createRedisAuthTokenSecret(
    config.environment,
    config.redisAuthToken || pulumi.output('')
  );

  const blnkRedisTokenSecret = createRedisAuthTokenSecret(
    config.environment,
    config.blnkRedisAuthToken || pulumi.output(''),
    true
  );

  // Create JWT secret if not provided
  let jwtSecretArn: pulumi.Output<string> | undefined;
  if (config.jwtSecret) {
    const jwtSecret = createJwtSecret(config.environment, config.jwtSecret);
    jwtSecretArn = jwtSecret.arn;
  }

  // Create master secret (consolidated)
  const masterSecret = createMasterSecret(config);
  createMasterSecretVersion(masterSecret, config);

  return {
    masterSecretArn: masterSecret.arn,
    masterSecretName: masterSecret.name,
    databasePasswordArn: dbPasswordSecret.arn,
    redisAuthTokenArn: redisTokenSecret.arn,
    blnkRedisAuthTokenArn: blnkRedisTokenSecret.arn,
    jwtSecretArn,
  };
}

/**
 * Get secret value by name (for use in other modules)
 */
export function getSecretValue(secretName: string): pulumi.Output<string> {
  return pulumi.output(
    aws.secretsmanager.getSecretVersion({
      secretId: secretName,
    })
  ).apply((version) => version.secretString);
}

/**
 * Get secret ARN by name
 */
export function getSecretArn(secretName: string): pulumi.Output<string> {
  return pulumi.output(
    aws.secretsmanager.getSecret({
      name: secretName,
    })
  ).apply((secret) => secret.arn);
}

/**
 * Create rotation schedule for secrets (prod only)
 */
export function createSecretRotation(
  environment: string,
  secret: aws.secretsmanager.Secret,
  rotationLambdaArn: string
): aws.secretsmanager.SecretRotation | undefined {
  if (environment !== 'prod') {
    return undefined;
  }

  return new aws.secretsmanager.SecretRotation(`ohi-${environment}-secret-rotation`, {
    secretId: secret.id,
    rotationLambdaArn: rotationLambdaArn,
    rotationRules: {
      automaticallyAfterDays: 90,
    },
  });
}

/**
 * Export secret ARN to SSM Parameter Store
 */
export function exportSecretArnToSsm(
  environment: string,
  secretType: string,
  secretArn: pulumi.Output<string>
): aws.ssm.Parameter {
  return new aws.ssm.Parameter(`ohi-${environment}-secret-arn-${secretType}`, {
    name: `/ohi/${environment}/secrets/${secretType}/arn`,
    type: 'String',
    value: secretArn,
    description: `Secret ARN for ${secretType} in ${environment}`,
    tags: { ...getResourceTags(environment, `secret-arn-${secretType}`) },
  });
}

/**
 * Get all secret ARNs for environment
 */
export function getAllSecretArns(environment: string): pulumi.Output<string[]> {
  return pulumi.output([
    getSecretArn(`ohi-${environment}-db-password`),
    getSecretArn(`ohi-${environment}-redis-token`),
    getSecretArn(`ohi-${environment}-blnk-redis-token`),
  ]);
}

/**
 * Create secret for API keys
 */
export function createApiKeysSecret(
  environment: string,
  apiKeys: Record<string, string>
): aws.secretsmanager.Secret {
  const secret = new aws.secretsmanager.Secret(`ohi-${environment}-api-keys`, {
    name: `ohi-${environment}-api-keys`,
    description: `API keys for OHI ${environment}`,
    recoveryWindowInDays: environment === 'prod' ? 30 : 7,
    tags: { ...getResourceTags(environment, 'api-keys-secret') },
  });

  new aws.secretsmanager.SecretVersion(`ohi-${environment}-api-keys-version`, {
    secretId: secret.id,
    secretString: pulumi.output(JSON.stringify(apiKeys)),
  });

  return secret;
}

/**
 * Validate secret configuration
 */
export function validateSecretsConfig(config: SecretsConfig): void {
  if (!config.environment) {
    throw new Error('Environment is required');
  }

  if (!config.databasePassword) {
    console.warn('Database password not provided, will need to be set manually');
  }

  if (!config.redisAuthToken) {
    console.warn('Redis auth token not provided, will need to be set manually');
  }
}
