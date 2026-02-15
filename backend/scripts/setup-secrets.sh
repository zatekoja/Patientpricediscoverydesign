#!/bin/bash
set -e

# Setup Secrets for Patient Price Discovery
# Usage: ./setup-secrets.sh [env]
# env defaults to 'dev'

ENV="${1:-dev}"
REGION="${AWS_REGION:-eu-west-1}"
PROJECT="ohi"

echo "========================================================"
echo "Setup Secrets for Environment: $ENV (Region: $REGION)"
echo "========================================================"

# List of secrets to configure
# Format: "SecretName|Description|AutoGenerate(true/false)"
SECRETS=(
  "postgres-password|Main PostgreSQL Database Password|true"
  "blnk-postgres-password|Blnk (Ledger) PostgreSQL Password|true"
  "typesense-api-key|Typesense Admin API Key|true"
  "geolocation-api-key|Google Maps/Geolocation API Key|false"
  "openai-api-key|OpenAI API Key|false"
  "calendly-api-key|Calendly API Key|false"
  "calendly-webhook-secret|Calendly Webhook Secret|false"
  "whatsapp-access-token|WhatsApp Cloud API Access Token|false"
  "flutterwave-secret-key|Flutterwave Secret Key|false"
  "flutterwave-webhook-secret|Flutterwave Webhook Secret|false"
  "redis-auth-token|Main Redis Auth Token|true"
  "blnk-redis-auth-token|Blnk Redis Auth Token|true"
  "provider-mongo-uri|MongoDB Atlas Connection String (provider-api)|false"
  "provider-llm-api-key|LLM API Key for Provider Document Parsing|false"
  "jwt-secret|JWT Signing Secret|true"
)

# Helper function to generate random string
generate_secret() {
  openssl rand -hex 16
}

# Helper function to check if secret exists
secret_exists() {
  local name=$1
  aws secretsmanager describe-secret --secret-id "$name" --region "$REGION" >/dev/null 2>&1
}

for item in "${SECRETS[@]}"; do
  # Split string by pipe
  IFS='|' read -r name description autogen <<< "$item"
  
  full_secret_name="${PROJECT}-${ENV}-${name}"
  
  echo ""
  echo "--------------------------------------------------------"
  echo "Configuring: $full_secret_name"
  echo "Description: $description"
  
  if secret_exists "$full_secret_name"; then
    echo "⚠️  Secret '$full_secret_name' already exists."
    read -p "Do you want to update it? (y/N): " update_choice
    if [[ ! "$update_choice" =~ ^[Yy]$ ]]; then
      echo "Skipping..."
      continue
    fi
  fi
  
  value=""
  
  # If autogen is true, ask if user wants to auto-generate
  if [ "$autogen" == "true" ]; then
    read -p "Auto-generate value? (Y/n): " gen_choice
    if [[ "$gen_choice" =~ ^[Nn]$ ]]; then
      read -s -p "Enter value: " value
      echo ""
    else
      value=$(generate_secret)
      echo "Generated value: $value"
    fi
  else
    # Manual entry required
    read -s -p "Enter value for $name: " value
    echo ""
  fi
  
  if [ -z "$value" ]; then
    echo "❌ Value cannot be empty. Skipping."
    continue
  fi
  
  # Create or Update
  if secret_exists "$full_secret_name"; then
    echo "Updating secret..."
    aws secretsmanager put-secret-value \
      --secret-id "$full_secret_name" \
      --secret-string "$value" \
      --region "$REGION" >/dev/null
  else
    echo "Creating secret..."
    aws secretsmanager create-secret \
      --name "$full_secret_name" \
      --description "$description" \
      --secret-string "$value" \
      --tags Key=Project,Value=open-health-initiative Key=Environment,Value="$ENV" Key=ManagedBy,Value=script \
      --region "$REGION" >/dev/null
  fi
  
  echo "✅ Secret configured."
done

echo ""
echo "========================================================"
echo "All secrets processed."
echo "========================================================"
