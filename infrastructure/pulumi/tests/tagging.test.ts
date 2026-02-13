/**
 * Test: Tagging Strategy
 * 
 * Requirements from V1_DEPLOYMENT_ARCHITECTURE.md:
 * - All resources MUST have 11 required tags
 * - Tags must follow super strict enforcement for shared AWS account
 * - Resource naming must follow convention: {project}-{environment}-{service}-{resource-type}
 */

import * as pulumi from '@pulumi/pulumi';

describe('Tagging Strategy', () => {
  describe('Required Tags', () => {
    it('should define all 11 required tags', () => {
      // RED: This test will fail until we implement the tagging module
      const { requiredTags } = require('../src/tagging');
      
      expect(requiredTags).toBeDefined();
      expect(requiredTags).toHaveProperty('Project');
      expect(requiredTags).toHaveProperty('Environment');
      expect(requiredTags).toHaveProperty('Owner');
      expect(requiredTags).toHaveProperty('CostCenter');
      expect(requiredTags).toHaveProperty('Service');
      expect(requiredTags).toHaveProperty('ManagedBy');
      expect(requiredTags).toHaveProperty('CreatedBy');
      expect(requiredTags).toHaveProperty('CreatedDate');
      expect(requiredTags).toHaveProperty('DataClassification');
      expect(requiredTags).toHaveProperty('BackupPolicy');
      expect(requiredTags).toHaveProperty('Compliance');
    });

    it('should have correct default values for project-wide tags', () => {
      const { requiredTags } = require('../src/tagging');
      
      expect(requiredTags.Project).toBe('open-health-initiative');
      expect(requiredTags.Owner).toBe('platform-team');
      expect(requiredTags.CostCenter).toBe('ohi-infrastructure');
      expect(requiredTags.ManagedBy).toBe('pulumi');
    });

    it('should set CreatedDate to current date in YYYY-MM-DD format', () => {
      const { requiredTags } = require('../src/tagging');
      
      const datePattern = /^\d{4}-\d{2}-\d{2}$/;
      expect(requiredTags.CreatedDate).toMatch(datePattern);
    });
  });

  describe('Service-Specific Tags', () => {
    it('should define service tags for all microservices', () => {
      const { serviceTags } = require('../src/tagging');
      
      expect(serviceTags).toContain('api');
      expect(serviceTags).toContain('graphql');
      expect(serviceTags).toContain('sse');
      expect(serviceTags).toContain('provider-api');
      expect(serviceTags).toContain('reindexer');
      expect(serviceTags).toContain('blnk-api');
      expect(serviceTags).toContain('blnk-worker');
      expect(serviceTags).toContain('frontend');
    });

    it('should define service tags for infrastructure services', () => {
      const { serviceTags } = require('../src/tagging');
      
      expect(serviceTags).toContain('postgres-primary');
      expect(serviceTags).toContain('postgres-read-replica');
      expect(serviceTags).toContain('redis-cache');
      expect(serviceTags).toContain('redis-blnk');
      expect(serviceTags).toContain('mongodb-provider');
      expect(serviceTags).toContain('clickhouse');
      expect(serviceTags).toContain('signoz-collector');
    });
  });

  describe('Resource Naming Convention', () => {
    it('should generate correct resource name', () => {
      const { generateResourceName } = require('../src/tagging');
      
      const name = generateResourceName('prod', 'api', 'ecs-service');
      expect(name).toBe('ohi-prod-api-ecs-service');
    });

    it('should handle different environments', () => {
      const { generateResourceName } = require('../src/tagging');
      
      expect(generateResourceName('dev', 'api', 'task')).toBe('ohi-dev-api-task');
      expect(generateResourceName('staging', 'graphql', 'alb')).toBe('ohi-staging-graphql-alb');
      expect(generateResourceName('prod', 'postgres', 'primary')).toBe('ohi-prod-postgres-primary');
    });
  });

  describe('Tag Transformation', () => {
    it('should create resource transformation function', () => {
      const { applyDefaultTags } = require('../src/tagging');
      
      expect(applyDefaultTags).toBeDefined();
      expect(typeof applyDefaultTags).toBe('function');
    });

    it('should apply default tags to resource', () => {
      const { applyDefaultTags } = require('../src/tagging');
      
      const transformation = applyDefaultTags('dev', 'api');
      expect(transformation).toBeDefined();
      expect(typeof transformation).toBe('function');
    });

    it('should merge custom tags with default tags', () => {
      const { mergeTags } = require('../src/tagging');
      
      const defaultTags = { Project: 'ohi', Environment: 'dev' };
      const customTags = { Service: 'api', CustomTag: 'value' };
      
      const merged = mergeTags(defaultTags, customTags);
      
      expect(merged).toEqual({
        Project: 'ohi',
        Environment: 'dev',
        Service: 'api',
        CustomTag: 'value',
      });
    });

    it('should not allow overriding required tags', () => {
      const { mergeTags } = require('../src/tagging');
      
      const defaultTags = { Project: 'open-health-initiative', Environment: 'prod' };
      const customTags = { Project: 'hacker-attempt', Service: 'api' };
      
      const merged = mergeTags(defaultTags, customTags);
      
      // Default tags should win
      expect(merged.Project).toBe('open-health-initiative');
      expect(merged.Environment).toBe('prod');
    });
  });

  describe('Data Classification', () => {
    it('should define valid data classification values', () => {
      const { dataClassifications } = require('../src/tagging');
      
      expect(dataClassifications).toContain('public');
      expect(dataClassifications).toContain('internal');
      expect(dataClassifications).toContain('confidential');
      expect(dataClassifications).toContain('pii');
    });
  });

  describe('Backup Policy', () => {
    it('should define valid backup policy values', () => {
      const { backupPolicies } = require('../src/tagging');
      
      expect(backupPolicies).toContain('daily');
      expect(backupPolicies).toContain('weekly');
      expect(backupPolicies).toContain('none');
    });
  });

  describe('Compliance Tags', () => {
    it('should define valid compliance values', () => {
      const { complianceTags } = require('../src/tagging');
      
      expect(complianceTags).toContain('hipaa');
      expect(complianceTags).toContain('gdpr');
      expect(complianceTags).toContain('none');
    });
  });
});
