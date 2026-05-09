import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { grantPermission } from "@/api/tools";
import { toast } from "sonner";

interface PermissionPromptProps {
  onPermissionGranted: (duration: "once" | "session") => void;
  onPermissionDenied: () => void;
  toolName: string;
  path: string;
  originalCommand: string;
}

export function PermissionPrompt({
  onPermissionGranted,
  onPermissionDenied,
  toolName,
  path,
  originalCommand,
}: PermissionPromptProps) {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);

  const handlePermissionResponse = async (response: "once" | "session" | "no") => {
    setIsLoading(true);
    try {
      if (response === "no") {
        onPermissionDenied();
        toast.error(t("permissionDenied", { path }));
        return;
      }

      // Grant permission via API
      await grantPermission(path, originalCommand, response);

      // Notify the caller to update state
      onPermissionGranted(response);
      toast.success(
        response === "once"
          ? t("permissionGrantedOnce", { path })
          : t("permissionGrantedSession", { path })
      );
    } catch (error) {
      console.error("Permission error:", error);
      toast.error(t("permissionError"));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-border/10 border border-border/20 rounded-lg p-4 mb-4">
      <div className="mb-3">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0">
            {/* Warning icon */}
            <div className="h-8 w-8 flex items-center justify-center bg-yellow-100 text-yellow-800 rounded-full">
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
          </div>
          <div className="flex-1">
            <h3 className="font-medium text-yellow-800">{t("permissionRequiredTitle")}</h3>
            <p className="mt-1 text-sm text-yellow-700">
              {t("permissionRequiredMessage", {
                tool: toolName,
                path,
              })}
            </p>
            {originalCommand && (
              <div className="mt-2">
                <p className="font-medium text-yellow-800">{t("commandToExecute")}</p>
                <pre className="mt-1 bg-yellow-50 p-2 rounded text-xs font-mono break-all">
                  {originalCommand}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
      <div className="flex justify-end space-x-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() => handlePermissionResponse("no")}
          disabled={isLoading}
        >
          {t("permissionDeny")}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => handlePermissionResponse("once")}
          disabled={isLoading}
        >
          {t("permissionOnce")}
        </Button>
        <Button
          variant="default"
          size="sm"
          onClick={() => handlePermissionResponse("session")}
          disabled={isLoading}
        >
          {t("permissionSession")}
        </Button>
      </div>
    </div>
  );
}