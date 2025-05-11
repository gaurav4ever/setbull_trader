const BASE_URL = '/api/v1/groups';

/**
 * Makes an HTTP request with error handling.
 * @param {string} url
 * @param {RequestInit} [options]
 * @returns {Promise<any>}
 */
async function request(url, options = {}) {
    try {
        const response = await fetch(url, {
            ...options,
            mode: 'cors',
            credentials: 'omit',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
                ...(options.headers || {})
            }
        });
        if (!response.ok) {
            try {
                const errorData = await response.json();
                throw new Error(errorData.error || `Request failed with status ${response.status}`);
            } catch (parseError) {
                throw new Error(`Request failed with status ${response.status}: ${response.statusText}`);
            }
        }
        if (response.status === 204) {
            return { success: true }
        }
        return await response.json();
    } catch (error) {
        throw error;
    }
}

/**
 * List all stock groups.
 * @returns {Promise<any>}
 */
export async function listGroups() {
    return request(BASE_URL);
}

/**
 * Create a new stock group.
 * @param {string} entryType
 * @param {string[]} stockIds
 * @returns {Promise<any>}
 */
export async function createGroup(entryType, stockIds) {
    return request(BASE_URL, {
        method: 'POST',
        body: JSON.stringify({ entryType, stockIds })
    });
}

/**
 * Get a stock group by ID.
 * @param {string} id
 * @returns {Promise<any>}
 */
export async function getGroup(id) {
    return request(`${BASE_URL}/${id}`);
}

/**
 * Edit a stock group.
 * @param {string} id
 * @param {string[]} stockIds
 * @returns {Promise<any>}
 */
export async function editGroup(id, stockIds) {
    return request(`${BASE_URL}/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ stockIds })
    });
}

/**
 * Delete a stock group.
 * @param {string} id
 * @returns {Promise<any>}
 */
export async function deleteGroup(id) {
    return request(`${BASE_URL}/${id}`, { method: 'DELETE' });
}

/**
 * Execute a stock group.
 * @param {string} id
 * @returns {Promise<any>}
 */
export async function executeGroup(id) {
    return request(`${BASE_URL}/${id}/execute`, { method: 'POST' });
} 