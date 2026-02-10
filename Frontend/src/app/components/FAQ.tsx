import { useState } from "react";
import { ChevronDown, ChevronUp, X } from "lucide-react";

interface FAQItem {
  question: string;
  answer: string;
}

interface FAQProps {
  isOpen: boolean;
  onClose: () => void;
}

const faqData: FAQItem[] = [
  {
    question: "Who are we?",
    answer: "From a bustling Lagos market to a quiet, distant village in Kaduna, a universal worry haunts us all: access to reliable healthcare. For too long, Nigerians have navigated a health landscape riddled with uncertainty. The search for genuine medication often feels like a gamble, with a fear of counterfeit drugs lurking on pharmacy shelves. Finding a specialist means endless referrals and frustrating trips across town, only to face a waitlist and unexpected bills. When an emergency strikes, the critical question isn't just \"Can I get to a hospital?\" but \"Which hospital can I trust, and can I afford it?\"\n\nThis is the daily burden carried by millions: the mother struggling to find the exact, approved vaccine for her child; the elderly parent needing consistent, affordable chronic medication; the young professional trying to compare the cost of a necessary surgery without being extorted. These are not just inconveniences; they are life-and-death decisions made without the benefit of clear, reliable information.\n\nWe understand this struggle because we are part of it. That is why we built the Open Health Initiative (OHI/OH!), a digital lifeline designed to bring clarity, trust, and affordability back to Nigerian healthcare. Imagine a world where the power to verify, compare, and connect to legitimate health services is simply a tap away. A world where you know the price of your medication before you leave home and the quality of your hospital before you arrive. OH! is your personal, pocket-sized health navigator with a transparent, verified network of medications, hospital services, and their true costs. This isn't just an app; it's the trusted, 'surest' path to better health for every Nigerian.",
  },
  {
    question: "How to help us?",
    answer: "We've designed our platform with simplicity in mind, making it incredibly easy for healthcare facilities to join our network and help Nigerian patients access transparent pricing information.\n\nGetting Started is Simple:\nAll your facility needs is a price and service list—we accept Excel files, CSV, or Word documents. Our system is built to ingest all three formats seamlessly, so you can work with what you already have. No need to recreate your documentation or learn new systems.\n\nData Provider Partnership:\nTo maintain real-time accuracy and ensure our information stays current, facilities need to work with a service data provider. We're flexible and happy to collaborate with your existing data provider. However, we strongly recommend our trusted partner, MEGALEK, who has committed to strict Service Level Agreements (SLAs) that guarantee reliability, accuracy, and timely updates. This partnership ensures that patients always see the most current pricing and availability.\n\nOnboarding in Minutes:\nOur streamlined onboarding process means your facility can be live on the OH! platform in just minutes. Upload your price list, connect with your data provider, and you're ready to reach thousands of patients searching for transparent, trustworthy healthcare options. By joining us, you're not just listing your services—you're becoming part of a movement to transform Nigerian healthcare through transparency and trust.\n\nReady to get started? Contact our onboarding team, and we'll guide you through every step.",
  },
  {
    question: "What's coming next?",
    answer: "We're constantly working on new features and improvements. Stay tuned for updates on enhanced search capabilities, more healthcare providers, and better user experience.",
  },
  {
    question: "How to reach out to us?",
    answer: "You can reach out to us through our contact form, email us directly, or connect with us on social media. We're always happy to hear from you and answer any questions you may have.",
  },
];

export function FAQ({ isOpen, onClose }: FAQProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(0);

  const toggleExpanded = (index: number) => {
    setExpandedIndex(expandedIndex === index ? null : index);
  };

  return (
    <>
      {/* FAQ Modal */}
      {isOpen && (
        <div className="fixed inset-0 bg-transparent flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between p-6 border-b border-gray-200">
              <h2 className="text-2xl font-bold text-gray-900">Frequently Asked Questions</h2>
              <button
                onClick={onClose}
                className="p-2 hover:bg-gray-100 rounded-full transition-colors"
              >
                <X className="w-6 h-6 text-gray-500" />
              </button>
            </div>

            {/* FAQ Items */}
            <div className="overflow-y-auto p-6 space-y-3">
              {faqData.map((faq, index) => (
                <div
                  key={index}
                  className="border border-gray-200 rounded-xl overflow-hidden transition-all duration-200"
                >
                  {/* Question */}
                  <button
                    onClick={() => toggleExpanded(index)}
                    className="w-full flex items-center justify-between p-4 bg-white hover:bg-gray-50 transition-colors text-left"
                  >
                    <span className="font-semibold text-gray-900 text-lg pr-4">{faq.question}</span>
                    {expandedIndex === index ? (
                      <ChevronUp className="w-5 h-5 text-gray-500 flex-shrink-0" />
                    ) : (
                      <ChevronDown className="w-5 h-5 text-gray-500 flex-shrink-0" />
                    )}
                  </button>

                  {/* Answer */}
                  {expandedIndex === index && (
                    <div className="px-4 pb-4 pt-2 bg-gray-50">
                      <p className="text-gray-700 leading-relaxed whitespace-pre-line">{faq.answer}</p>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
