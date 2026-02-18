#!/bin/bash

echo "🦞 PicoClaw Deployment Helper"
echo "=============================="
echo ""
echo "Choose deployment platform:"
echo ""
echo "1) Railway.app (Recommended - $5/month free credit)"
echo "2) Render.com (Free - 750 hours/month)"
echo "3) Fly.io (Free - 3 VMs)"
echo "4) Show instructions only"
echo ""
read -p "Enter choice [1-4]: " choice

case $choice in
    1)
        echo ""
        echo "📦 Deploying to Railway..."
        echo ""
        
        # Check if Railway CLI is installed
        if ! command -v railway &> /dev/null; then
            echo "Installing Railway CLI..."
            npm install -g @railway/cli
        fi
        
        # Login if needed
        railway login
        
        # Initialize project
        echo ""
        echo "Run these commands:"
        echo "  railway init"
        echo "  railway variables set TELEGRAM_BOT_TOKEN=YOUR_TOKEN"
        echo "  railway variables set ZHIPU_API_KEY=YOUR_KEY"
        echo "  railway up"
        ;;
    2)
        echo ""
        echo "📦 Deploying to Render..."
        echo ""
        echo "Steps:"
        echo "1. Go to https://render.com"
        echo "2. Create account"
        echo "3. New → Web Service"
        echo "4. Connect GitHub repo"
        echo "5. Select Docker environment"
        echo "6. Add environment variables"
        echo "7. Deploy"
        ;;
    3)
        echo ""
        echo "📦 Deploying to Fly.io..."
        echo ""
        
        # Check if Fly CLI is installed
        if ! command -v fly &> /dev/null; then
            echo "Installing Fly CLI..."
            brew install flyctl
        fi
        
        echo "Run these commands:"
        echo "  fly auth login"
        echo "  fly secrets set TELEGRAM_BOT_TOKEN=YOUR_TOKEN"
        echo "  fly secrets set ZHIPU_API_KEY=YOUR_KEY"
        echo "  fly launch"
        echo "  fly deploy"
        ;;
    4)
        echo ""
        echo "📖 See ~/.picoclaw/workspace/DEPLOYMENT.md for detailed instructions"
        ;;
    *)
        echo "Invalid choice"
        ;;
esac
