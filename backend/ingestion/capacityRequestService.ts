import crypto from 'crypto';
import { IDocumentStore } from '../interfaces/IDocumentStore';
import { FacilityProfile } from '../types/FacilityProfile';
import { FacilityProfileService } from './facilityProfileService';
import {
  recordCapacityRequest,
  recordCapacityRequestJob,
  recordCapacityTokenConsumed,
  recordCapacityTokenIssued,
} from '../observability/metrics';

export type CapacityChannel = 'email' | 'whatsapp';

export interface CapacityRequestToken {
  id: string;
  facilityId: string;
  channel: CapacityChannel;
  recipient: string;
  createdAt: string;
  expiresAt: string;
  usedAt?: string;
}

export interface EmailSender {
  send(to: string, subject: string, html: string, text?: string): Promise<void>;
}

export interface WhatsAppSender {
  send(to: string, message: string, link?: string): Promise<void>;
}

export interface CapacityRequestServiceOptions {
  facilityProfileService: FacilityProfileService;
  tokenStore: IDocumentStore<CapacityRequestToken>;
  publicBaseUrl: string;
  tokenTTLMinutes?: number; // Default: 120 minutes, can be overridden per facility
  emailSender?: EmailSender;
  whatsappSender?: WhatsAppSender;
  emailTemplate?: (facilityName: string, link: string) => string; // Custom email template
}

export class CapacityRequestService {
  private running = false;
  private timer?: NodeJS.Timeout;

  constructor(private options: CapacityRequestServiceOptions) {}

  start(intervalMs: number): void {
    if (intervalMs <= 0) {
      return;
    }
    if (this.timer) {
      clearInterval(this.timer);
    }
    this.timer = setInterval(() => {
      this.runOnce().catch((error) => {
        console.error('Capacity request job failed:', error);
      });
    }, intervalMs);
  }

  stop(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = undefined;
    }
  }

  async runOnce(): Promise<void> {
    if (this.running) {
      return;
    }
    this.running = true;
    const startTime = Date.now();
    let success = false;
    try {
      const facilities = await this.listAllFacilities();
      for (const facility of facilities) {
        try {
          await this.requestForFacility(facility);
        } catch (error) {
          console.error('Capacity request failed for facility:', facility.id, error);
        }
      }
      success = true;
    } finally {
      recordCapacityRequestJob({ success, durationMs: Date.now() - startTime });
      this.running = false;
    }
  }

  async consumeToken(rawToken: string): Promise<CapacityRequestToken> {
    const tokenHash = hashToken(rawToken);
    const record = await this.options.tokenStore.get(tokenHash);
    if (!record) {
      throw new Error('Invalid token');
    }
    if (record.usedAt) {
      throw new Error('Token already used');
    }
    const now = new Date();
    if (new Date(record.expiresAt).getTime() < now.getTime()) {
      throw new Error('Token expired');
    }

    record.usedAt = now.toISOString();
    await this.options.tokenStore.put(tokenHash, record, { usedAt: record.usedAt });
    recordCapacityTokenConsumed(record.channel);
    return record;
  }

  async sendSingleRequest(facilityId: string, channel?: CapacityChannel): Promise<void> {
    const facility = await this.options.facilityProfileService.getProfile(facilityId);
    if (!facility) {
      throw new Error('Facility not found');
    }
    if (channel) {
      if (channel === 'email') {
        const emailRecipient = facility.email?.trim();
        if (!emailRecipient || !this.options.emailSender) {
          throw new Error('Email sender not configured or facility email missing');
        }
        await this.sendRequest(facility, 'email', emailRecipient);
        return;
      }
      const phoneRecipient = facility.phoneNumber?.trim();
      if (!phoneRecipient || !this.options.whatsappSender) {
        throw new Error('WhatsApp sender not configured or facility phone missing');
      }
      await this.sendRequest(facility, 'whatsapp', phoneRecipient);
      return;
    }

    await this.requestForFacility(facility);
  }

  private async requestForFacility(facility: FacilityProfile): Promise<void> {
    const emailRecipient = facility.email?.trim();
    const phoneRecipient = facility.phoneNumber?.trim();

    if (emailRecipient && this.options.emailSender) {
      await this.sendRequest(facility, 'email', emailRecipient);
    }
    if (phoneRecipient && this.options.whatsappSender) {
      await this.sendRequest(facility, 'whatsapp', phoneRecipient);
    }
  }

  private async sendRequest(
    facility: FacilityProfile,
    channel: CapacityChannel,
    recipient: string,
  ): Promise<void> {
    const active = await this.findActiveToken(facility.id, channel, recipient);
    if (active) {
      return;
    }

    // Get TTL from facility metadata or use default
    const facilityTTL = facility.metadata?.capacityTokenTTLMinutes;
    const tokenTTL = facilityTTL ?? this.options.tokenTTLMinutes ?? 120;

    const { rawToken, tokenRecord } = createTokenRecord(
      facility.id,
      channel,
      recipient,
      tokenTTL
    );
    await this.options.tokenStore.put(tokenRecord.id, tokenRecord, {
      facilityId: facility.id,
      channel,
      recipient,
    });
    recordCapacityTokenIssued(channel);

    const link = this.buildFormLink(rawToken);
    if (channel === 'email' && this.options.emailSender) {
      try {
        const subject = `Update ${facility.name} capacity status`;
        // Use custom template if provided, otherwise use default
        const body = this.options.emailTemplate
          ? this.options.emailTemplate(facility.name, link)
          : buildEmailBody(facility.name, link);
        await this.options.emailSender.send(recipient, subject, body, stripHtml(body));
        recordCapacityRequest({ channel, success: true });
      } catch (error) {
        recordCapacityRequest({ channel, success: false });
        throw error;
      }
    }
    if (channel === 'whatsapp' && this.options.whatsappSender) {
      try {
        const message = `Please update capacity for ${facility.name}: ${link}`;
        await this.options.whatsappSender.send(recipient, message, link);
        recordCapacityRequest({ channel, success: true });
      } catch (error) {
        recordCapacityRequest({ channel, success: false });
        throw error;
      }
    }
  }

  private buildFormLink(rawToken: string): string {
    const base = this.options.publicBaseUrl.replace(/\/+$/, '');
    return `${base}/api/v1/capacity/form/${encodeURIComponent(rawToken)}`;
  }

  private async listAllFacilities(): Promise<FacilityProfile[]> {
    const facilities: FacilityProfile[] = [];
    const pageSize = 200;
    let offset = 0;
    for (;;) {
      const page = await this.options.facilityProfileService.listProfiles(pageSize, offset);
      if (!page.length) {
        break;
      }
      facilities.push(...page);
      offset += page.length;
      if (page.length < pageSize) {
        break;
      }
    }
    return facilities;
  }

  private async findActiveToken(
    facilityId: string,
    channel: CapacityChannel,
    recipient: string
  ): Promise<CapacityRequestToken | null> {
    const tokens = await this.options.tokenStore.query(
      { facilityId, channel, recipient },
      { limit: 10, sortBy: 'createdAt', sortOrder: 'desc' }
    );
    const now = Date.now();
    for (const token of tokens) {
      if (token.usedAt) {
        continue;
      }
      if (new Date(token.expiresAt).getTime() < now) {
        continue;
      }
      return token;
    }
    return null;
  }
}

export class SesEmailSender implements EmailSender {
  constructor(private region: string, private fromAddress: string) {}

  async send(to: string, subject: string, html: string, text?: string): Promise<void> {
    const aws = await import('aws-sdk');
    const SES = (aws as any).SES;
    const ses = new SES({ region: this.region });
    await ses
      .sendEmail({
        Source: this.fromAddress,
        Destination: { ToAddresses: [to] },
        Message: {
          Subject: { Data: subject },
          Body: {
            Html: { Data: html },
            Text: { Data: text || stripHtml(html) },
          },
        },
      })
      .promise();
  }
}

export class WhatsAppCloudSender implements WhatsAppSender {
  constructor(
    private accessToken: string,
    private phoneNumberId: string,
    private templateName?: string,
    private templateLang: string = 'en_US',
  ) {}

  async send(to: string, message: string, link?: string): Promise<void> {
    const endpoint = `https://graph.facebook.com/v20.0/${this.phoneNumberId}/messages`;
    const payload = this.templateName
      ? {
          messaging_product: 'whatsapp',
          to,
          type: 'template',
          template: {
            name: this.templateName,
            language: { code: this.templateLang },
            components: link
              ? [
                  {
                    type: 'body',
                    parameters: [
                      { type: 'text', text: message },
                      { type: 'text', text: link },
                    ],
                  },
                ]
              : undefined,
          },
        }
      : {
          messaging_product: 'whatsapp',
          to,
          type: 'text',
          text: { body: message },
        };

    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.accessToken}`,
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`WhatsApp API error (${response.status}): ${text}`);
    }
  }
}

function createTokenRecord(
  facilityId: string,
  channel: CapacityChannel,
  recipient: string,
  ttlMinutes: number
): { rawToken: string; tokenRecord: CapacityRequestToken } {
  const rawToken = crypto.randomBytes(32).toString('base64url');
  const tokenHash = hashToken(rawToken);
  const now = new Date();
  const expiresAt = new Date(now.getTime() + ttlMinutes * 60 * 1000);
  const tokenRecord: CapacityRequestToken = {
    id: tokenHash,
    facilityId,
    channel,
    recipient,
    createdAt: now.toISOString(),
    expiresAt: expiresAt.toISOString(),
  };
  return { rawToken, tokenRecord };
}

function hashToken(rawToken: string): string {
  return crypto.createHash('sha256').update(rawToken).digest('hex');
}

function buildEmailBody(facilityName: string, link: string): string {
  return `
    <p>Hello ${facilityName} team,</p>
    <p>Please update your current capacity and wait time using the secure link below:</p>
    <p><a href="${link}">${link}</a></p>
    <p>This link expires soon for security.</p>
  `;
}

function stripHtml(html: string): string {
  return html.replace(/<[^>]+>/g, '').trim();
}
