import { useState, type FormEvent } from "react";
import { Star, X } from "lucide-react";
import { api } from "../../lib/api";

const ratingLabels = ["", "Poor", "Fair", "Good", "Great", "Excellent"];

export function FeedbackTab() {
  const [open, setOpen] = useState(false);
  const [rating, setRating] = useState(0);
  const [message, setMessage] = useState("");
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<"idle" | "submitting" | "success" | "error">("idle");

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (rating < 1) {
      setStatus("error");
      return;
    }

    setStatus("submitting");
    try {
      await api.submitFeedback({
        rating,
        message: message.trim() || undefined,
        email: email.trim() || undefined,
        page: window.location.pathname,
      });
      setStatus("success");
      setMessage("");
      setEmail("");
      setRating(0);
    } catch {
      setStatus("error");
    }
  };

  return (
    <div className="fixed right-3 bottom-3 z-40 flex flex-col items-end gap-2">
      {open && (
        <div
          className="w-[320px] max-w-[92vw] rounded-2xl border border-gray-200 bg-white shadow-2xl p-4"
          role="dialog"
          aria-label="Share feedback"
        >
          <div className="flex items-start justify-between gap-4">
            <div>
              <h3 className="text-sm font-semibold text-gray-900">Quick feedback</h3>
              <p className="text-xs text-gray-500">Share in under a minute.</p>
            </div>
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="text-gray-400 hover:text-gray-600"
              aria-label="Close feedback form"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <form onSubmit={handleSubmit} className="mt-4 space-y-3">
            <div>
              <p className="text-xs font-medium text-gray-700">How was your experience?</p>
              <div className="mt-2 flex items-center gap-2">
                {[1, 2, 3, 4, 5].map((value) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => {
                      setRating(value);
                      if (status === "error") {
                        setStatus("idle");
                      }
                    }}
                    className={`flex h-8 w-8 items-center justify-center rounded-full border ${
                      rating >= value
                        ? "border-yellow-400 bg-yellow-50 text-yellow-500"
                        : "border-gray-200 text-gray-400"
                    }`}
                    aria-label={`Rate ${value} out of 5`}
                  >
                    <Star className="h-4 w-4" />
                  </button>
                ))}
                {rating > 0 && (
                  <span className="text-xs text-gray-500">{ratingLabels[rating]}</span>
                )}
              </div>
            </div>

            <div>
              <label className="text-xs font-medium text-gray-700">What could be better? (optional)</label>
              <textarea
                rows={3}
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                className="mt-2 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Tell us what to improve"
              />
            </div>

            <div>
              <label className="text-xs font-medium text-gray-700">Email (optional)</label>
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="mt-2 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="name@email.com"
              />
            </div>

            {status === "error" && (
              <p className="text-xs text-red-600">
                Please add a rating. If the issue persists, try again in a minute.
              </p>
            )}
            {status === "success" && (
              <p className="text-xs text-green-600">Thanks! Your feedback was received.</p>
            )}

            <button
              type="submit"
              disabled={status === "submitting"}
              className="w-full rounded-lg bg-blue-600 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-400"
            >
              {status === "submitting" ? "Sending..." : "Send feedback"}
            </button>

            <p className="text-[11px] text-gray-400">
              No sensitive data, please. We only use this to improve the product.
            </p>
          </form>
        </div>
      )}

      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="rounded-full bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-lg hover:bg-blue-700"
        aria-haspopup="dialog"
        aria-expanded={open}
      >
        Feedback
      </button>
    </div>
  );
}
