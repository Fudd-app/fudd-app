import React from "react";
import LandingHeader from "@/components/landing/LandingHeader";
import LandingHero from "@/components/landing/LandingHero";
import LandingFeatures from "@/components/landing/LandingFeatures";
import LandingBenefits from "@/components/landing/LandingBenefits";
import LandingPricing from "@/components/landing/LandingPricing";
import LandingFAQ from "@/components/landing/LandingFAQ";
import LandingFinalCta from "@/components/landing/LandingFinalCta";
import LandingFooter from "@/components/landing/LandingFooter";
import "@/components/landing/landing.css";

export default function LandingPage() {
  return (
    <div id="inicio" className="landing font-inter antialiased">
      <LandingHeader />
      <main>
        <LandingHero />
        <LandingFeatures />
        <LandingBenefits />
        <LandingPricing />
        <LandingFAQ />
        <LandingFinalCta />
      </main>
      <LandingFooter />
    </div>
  );
}
