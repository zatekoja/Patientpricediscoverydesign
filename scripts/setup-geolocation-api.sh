#!/bin/bash
# Setup geolocation API key from GCloud to Vault

set -e

PROJECT_ID=$(gcloud config get-value project)
API_KEY_ID="090917f5-3964-4d06-82a6-35090c2ded33"

echo "╔════════════════════════════════════════════════════════╗"
echo "║  GCloud Geolocation API Key Setup for Vault            ║"
echo "╚════════════════════════════════════════════════════════╝"

echo ""
echo "📋 Instructions to get the API key:"
echo "   1. Open GCloud Console: https://console.cloud.google.com"
echo "   2. Navigate to: APIs & Services > Credentials"
echo "   3. Find key: 'ppd-geolocation' (ID: $API_KEY_ID)"
echo "   4. Click on it to reveal the key string"
echo "   5. Copy the full key value"
echo ""

read -sp "🔑 Paste the API key: " GEOLOCATION_API_KEY
echo ""

if [ -z "$GEOLOCATION_API_KEY" ]; then
    echo "❌ Error: API key cannot be empty"
    exit 1
fi

echo "✓ Updating Vault with geolocation API key..."

# Update the api.json file with the actual key
cat > /Users/zatekoja/GolandProjects/Patientpricediscoverydesign/vault/init/api.json << EOF
{
  "OPENAI_API_KEY": "dev-openai",
  "GEOLOCATION_API_KEY": "$GEOLOCATION_API_KEY",
  "TYPESENSE_API_KEY": "xyz"
}
EOF

echo "✓ Updated vault/init/api.json"

echo ""
echo "🔄 Restarting Vault and API services..."
docker compose down vault vault-init
sleep 2
docker compose up -d vault vault-init
sleep 5
docker compose restart api

echo ""
echo "✅ Geolocation API key configured in Vault!"
echo "   • Location: secret/patient-price-discovery/api/GEOLOCATION_API_KEY"
echo "   • Verify with: docker exec ppd_vault vault kv get -mount=secret patient-price-discovery/api"
