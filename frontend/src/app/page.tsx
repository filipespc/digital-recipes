import Link from 'next/link';

export default function Home() {
  return (
    <div className="min-h-[calc(100vh-4rem)] bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="py-10">
          <div className="pb-4 border-b border-gray-200">
            <h1 className="text-3xl font-bold text-gray-900">Welcome to Digital Recipes</h1>
            <p className="mt-2 text-sm text-gray-600">
              AI-Powered Recipe Hub - Organize and manage your recipes
            </p>
          </div>
          
          <main className="mt-8">
            <div className="text-center">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">
                Welcome to Your Recipe Hub
              </h2>
              <p className="text-gray-600 mb-8">
                Start by viewing your recipe collection
              </p>
              <Link
                href="/recipes"
                className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg font-medium transition-colors"
              >
                View Recipes
              </Link>
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}
