// frontend/src/lib/actions/clickOutside.js

/**
 * Action to detect clicks outside of an element
 * @param {HTMLElement} node - The element to detect clicks outside of
 * @returns {object} - Svelte action object
 */
export function clickOutside(node) {
    const handleClick = (event) => {
        if (node && !node.contains(event.target) && !event.defaultPrevented) {
            node.dispatchEvent(new CustomEvent('click_outside', { detail: node }));
        }
    };

    document.addEventListener('click', handleClick, true);

    return {
        destroy() {
            document.removeEventListener('click', handleClick, true);
        }
    };
}