/**
 * Utilidades para manejo de autenticación con JWT y localStorage
 */

const TOKEN_KEY = "talentnest_token";

/**
 * Guardar el token JWT en localStorage
 */
export const saveToken = (token: string): void => {
    localStorage.setItem(TOKEN_KEY, token);
};

/**
 * Obtener el token JWT desde localStorage
 */
export const getToken = (): string | null => {
    return localStorage.getItem(TOKEN_KEY);
};

/**
 * Eliminar el token JWT de localStorage
 */
export const removeToken = (): void => {
    localStorage.removeItem(TOKEN_KEY);
};

/**
 * Verificar si el usuario está autenticado (tiene un token)
 */
export const isAuthenticated = (): boolean => {
    return getToken() !== null;
};

/**
 * Hacer logout: eliminar el token y redirigir al login
 */
export const logout = (): void => {
    removeToken();
    window.location.href = "/login";
};
