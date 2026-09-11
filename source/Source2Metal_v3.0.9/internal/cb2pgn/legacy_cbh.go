// Legacy CBH decoder ported from Dominik Klein's cbh2pgn 0.1 (2022).
// Original source is bundled under reference_cbh2pgn-0.1/; see THIRD_PARTY_NOTICES.txt.
//
// Important: classic CBH one-byte moves are NOT translated through the
// DEOBFUSCATE_2B table.  The exact reference algorithm first computes
//
//	token = (rawByte - processedMoves) mod 256
//
// and looks that token up directly in the per-piece token tables below.
// Only the two payload bytes following token 0x29 use DEOBFUSCATE_2B.
// This distinction is what the v0.1.1 real-world StrongGames2017 test exposed.
package cb2pgn

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	cbWQueen  = 1
	cbWKnight = 2
	cbWBishop = 3
	cbWRook   = 4
	cbBQueen  = 5
	cbBKnight = 6
	cbBBishop = 7
	cbBRook   = 8
	cbWKing   = 9
	cbBKing   = 10
	cbWPawn   = 11
	cbBPawn   = 12
)

type legacyTokenSpec struct{ kind, no, dx, dy, castle int }

var legacyOneByte = map[byte]legacyTokenSpec{
	0x00: {queen, 1, 6, 6, 0},    // CB_QUEEN_2_ENC
	0x01: {queen, 1, 0, 7, 0},    // CB_QUEEN_2_ENC
	0x02: {bishop, 0, 1, 1, 0},   // CB_BISHOP_1_ENC
	0x04: {queen, 2, 2, 6, 0},    // CB_QUEEN_3_ENC
	0x05: {rook, 1, 2, 0, 0},     // CB_ROOK_2_ENC
	0x06: {bishop, 0, 1, 7, 0},   // CB_BISHOP_1_ENC
	0x07: {knight, 1, -1, -2, 0}, // CB_KNIGHT_2_ENC
	0x08: {bishop, 1, 3, 3, 0},   // CB_BISHOP_2_ENC
	0x09: {pawn, 5, 0, 1, 0},     // CB_PAWN_F_ENC
	0x0A: {rook, 2, 0, 6, 0},     // CB_ROOK_3_ENC
	0x0B: {pawn, 3, 0, 2, 0},     // CB_PAWN_D_ENC
	0x0D: {queen, 2, 0, 4, 0},    // CB_QUEEN_3_ENC
	0x0E: {knight, 1, 1, 2, 0},   // CB_KNIGHT_2_ENC
	0x0F: {queen, 2, 0, 3, 0},    // CB_QUEEN_3_ENC
	0x10: {rook, 2, 4, 0, 0},     // CB_ROOK_3_ENC
	0x11: {queen, 1, 0, 4, 0},    // CB_QUEEN_2_ENC
	0x12: {pawn, 7, 0, 1, 0},     // CB_PAWN_H_ENC
	0x13: {pawn, 7, 1, 1, 0},     // CB_PAWN_H_ENC
	0x14: {rook, 1, 0, 1, 0},     // CB_ROOK_2_ENC
	0x15: {pawn, 4, 1, 1, 0},     // CB_PAWN_E_ENC
	0x16: {bishop, 1, 7, 1, 0},   // CB_BISHOP_2_ENC
	0x17: {pawn, 1, 0, 2, 0},     // CB_PAWN_B_ENC
	0x18: {queen, 0, 7, 1, 0},    // CB_QUEEN_1_ENC
	0x19: {pawn, 7, -1, 1, 0},    // CB_PAWN_H_ENC
	0x1A: {queen, 2, 0, 1, 0},    // CB_QUEEN_3_ENC
	0x1B: {rook, 2, 0, 4, 0},     // CB_ROOK_3_ENC
	0x1D: {queen, 1, 5, 0, 0},    // CB_QUEEN_2_ENC
	0x1F: {queen, 1, 4, 4, 0},    // CB_QUEEN_2_ENC
	0x20: {queen, 1, 2, 6, 0},    // CB_QUEEN_2_ENC
	0x21: {queen, 0, 4, 0, 0},    // CB_QUEEN_1_ENC
	0x23: {queen, 2, 0, 7, 0},    // CB_QUEEN_3_ENC
	0x24: {queen, 0, 6, 6, 0},    // CB_QUEEN_1_ENC
	0x26: {rook, 0, 3, 0, 0},     // CB_ROOK_1_ENC
	0x27: {knight, 2, 2, -1, 0},  // CB_KNIGHT_3_ENC
	0x28: {queen, 0, 3, 5, 0},    // CB_QUEEN_1_ENC
	0x2A: {queen, 1, 4, 4, 0},    // CB_QUEEN_2_ENC
	0x2B: {rook, 2, 0, 7, 0},     // CB_ROOK_3_ENC
	0x2C: {bishop, 0, 5, 3, 0},   // CB_BISHOP_1_ENC
	0x2D: {pawn, 0, 0, 1, 0},     // CB_PAWN_A_ENC
	0x2E: {rook, 0, 1, 0, 0},     // CB_ROOK_1_ENC
	0x2F: {queen, 0, 5, 3, 0},    // CB_QUEEN_1_ENC
	0x30: {rook, 0, 5, 0, 0},     // CB_ROOK_1_ENC
	0x31: {queen, 1, 0, 6, 0},    // CB_QUEEN_2_ENC
	0x32: {rook, 1, 6, 0, 0},     // CB_ROOK_2_ENC
	0x33: {pawn, 7, 0, 2, 0},     // CB_PAWN_H_ENC
	0x34: {knight, 1, 2, -1, 0},  // CB_KNIGHT_2_ENC
	0x35: {bishop, 1, 1, 7, 0},   // CB_BISHOP_2_ENC
	0x36: {pawn, 4, -1, 1, 0},    // CB_PAWN_E_ENC
	0x37: {bishop, 0, 7, 1, 0},   // CB_BISHOP_1_ENC
	0x38: {queen, 2, 3, 3, 0},    // CB_QUEEN_3_ENC
	0x39: {king, 0, 1, 1, 0},     // CB_KING_ENC
	0x3A: {pawn, 6, -1, 1, 0},    // CB_PAWN_G_ENC
	0x3B: {bishop, 2, 4, 4, 0},   // CB_BISHOP_3_ENC
	0x3D: {knight, 0, 1, 2, 0},   // CB_KNIGHT_1_ENC
	0x3E: {bishop, 2, 3, 5, 0},   // CB_BISHOP_3_ENC
	0x3F: {bishop, 1, 2, 2, 0},   // CB_BISHOP_2_ENC
	0x40: {queen, 2, 2, 2, 0},    // CB_QUEEN_3_ENC
	0x41: {bishop, 0, 4, 4, 0},   // CB_BISHOP_1_ENC
	0x42: {queen, 2, 0, 2, 0},    // CB_QUEEN_3_ENC
	0x43: {rook, 0, 0, 3, 0},     // CB_ROOK_1_ENC
	0x44: {queen, 1, 1, 1, 0},    // CB_QUEEN_2_ENC
	0x45: {bishop, 2, 3, 3, 0},   // CB_BISHOP_3_ENC
	0x46: {bishop, 2, 4, 4, 0},   // CB_BISHOP_3_ENC
	0x47: {king, 0, 7, 1, 0},     // CB_KING_ENC
	0x48: {queen, 0, 2, 6, 0},    // CB_QUEEN_1_ENC
	0x49: {king, 0, 0, 1, 0},     // CB_KING_ENC
	0x4A: {knight, 0, 2, -1, 0},  // CB_KNIGHT_1_ENC
	0x4B: {queen, 1, 7, 7, 0},    // CB_QUEEN_2_ENC
	0x4D: {queen, 0, 1, 1, 0},    // CB_QUEEN_1_ENC
	0x4E: {rook, 0, 0, 1, 0},     // CB_ROOK_1_ENC
	0x4F: {queen, 2, 4, 0, 0},    // CB_QUEEN_3_ENC
	0x50: {queen, 1, 0, 3, 0},    // CB_QUEEN_2_ENC
	0x51: {bishop, 2, 1, 1, 0},   // CB_BISHOP_3_ENC
	0x52: {rook, 1, 7, 0, 0},     // CB_ROOK_2_ENC
	0x53: {queen, 0, 0, 4, 0},    // CB_QUEEN_1_ENC
	0x54: {queen, 2, 3, 0, 0},    // CB_QUEEN_3_ENC
	0x55: {bishop, 0, 3, 5, 0},   // CB_BISHOP_1_ENC
	0x56: {bishop, 2, 5, 5, 0},   // CB_BISHOP_3_ENC
	0x57: {queen, 0, 7, 0, 0},    // CB_QUEEN_1_ENC
	0x58: {knight, 0, 2, 1, 0},   // CB_KNIGHT_1_ENC
	0x59: {queen, 2, 4, 4, 0},    // CB_QUEEN_3_ENC
	0x5A: {queen, 0, 6, 2, 0},    // CB_QUEEN_1_ENC
	0x5B: {queen, 1, 3, 5, 0},    // CB_QUEEN_2_ENC
	0x5C: {queen, 1, 1, 0, 0},    // CB_QUEEN_2_ENC
	0x5D: {king, 0, 1, 7, 0},     // CB_KING_ENC
	0x5E: {bishop, 1, 6, 6, 0},   // CB_BISHOP_2_ENC
	0x5F: {knight, 1, -2, 1, 0},  // CB_KNIGHT_2_ENC
	0x60: {queen, 1, 7, 1, 0},    // CB_QUEEN_2_ENC
	0x61: {rook, 0, 6, 0, 0},     // CB_ROOK_1_ENC
	0x62: {queen, 0, 4, 4, 0},    // CB_QUEEN_1_ENC
	0x63: {rook, 0, 0, 5, 0},     // CB_ROOK_1_ENC
	0x64: {pawn, 1, 0, 1, 0},     // CB_PAWN_B_ENC
	0x66: {bishop, 2, 2, 6, 0},   // CB_BISHOP_3_ENC
	0x67: {queen, 1, 1, 7, 0},    // CB_QUEEN_2_ENC
	0x68: {rook, 1, 0, 3, 0},     // CB_ROOK_2_ENC
	0x69: {rook, 2, 6, 0, 0},     // CB_ROOK_3_ENC
	0x6A: {queen, 2, 6, 2, 0},    // CB_QUEEN_3_ENC
	0x6B: {queen, 0, 0, 6, 0},    // CB_QUEEN_1_ENC
	0x6C: {queen, 2, 7, 7, 0},    // CB_QUEEN_3_ENC
	0x6D: {bishop, 1, 3, 5, 0},   // CB_BISHOP_2_ENC
	0x6E: {queen, 0, 4, 4, 0},    // CB_QUEEN_1_ENC
	0x6F: {rook, 0, 7, 0, 0},     // CB_ROOK_1_ENC
	0x70: {pawn, 1, 1, 1, 0},     // CB_PAWN_B_ENC
	0x71: {bishop, 1, 4, 4, 0},   // CB_BISHOP_2_ENC
	0x72: {queen, 2, 7, 0, 0},    // CB_QUEEN_3_ENC
	0x73: {bishop, 1, 5, 5, 0},   // CB_BISHOP_2_ENC
	0x74: {rook, 2, 5, 0, 0},     // CB_ROOK_3_ENC
	0x75: {knight, 1, -2, -1, 0}, // CB_KNIGHT_2_ENC
	0x76: {king, 0, 2, 0, 1},     // CB_KING_ENC
	0x77: {rook, 1, 0, 6, 0},     // CB_ROOK_2_ENC
	0x78: {bishop, 1, 7, 7, 0},   // CB_BISHOP_2_ENC
	0x79: {queen, 0, 1, 0, 0},    // CB_QUEEN_1_ENC
	0x7A: {queen, 2, 2, 0, 0},    // CB_QUEEN_3_ENC
	0x7B: {pawn, 2, 0, 1, 0},     // CB_PAWN_C_ENC
	0x7C: {bishop, 0, 6, 6, 0},   // CB_BISHOP_1_ENC
	0x7D: {pawn, 5, 1, 1, 0},     // CB_PAWN_F_ENC
	0x7E: {queen, 1, 6, 0, 0},    // CB_QUEEN_2_ENC
	0x7F: {queen, 0, 0, 5, 0},    // CB_QUEEN_1_ENC
	0x80: {queen, 1, 2, 2, 0},    // CB_QUEEN_2_ENC
	0x81: {rook, 2, 0, 1, 0},     // CB_ROOK_3_ENC
	0x82: {rook, 2, 0, 2, 0},     // CB_ROOK_3_ENC
	0x83: {queen, 1, 5, 5, 0},    // CB_QUEEN_2_ENC
	0x84: {pawn, 4, 0, 1, 0},     // CB_PAWN_E_ENC
	0x85: {pawn, 2, -1, 1, 0},    // CB_PAWN_C_ENC
	0x86: {queen, 2, 1, 7, 0},    // CB_QUEEN_3_ENC
	0x87: {queen, 2, 5, 5, 0},    // CB_QUEEN_3_ENC
	0x88: {rook, 0, 4, 0, 0},     // CB_ROOK_1_ENC
	0x89: {knight, 1, 1, -2, 0},  // CB_KNIGHT_2_ENC
	0x8B: {rook, 1, 3, 0, 0},     // CB_ROOK_2_ENC
	0x8C: {queen, 2, 4, 4, 0},    // CB_QUEEN_3_ENC
	0x8D: {queen, 0, 0, 7, 0},    // CB_QUEEN_1_ENC
	0x8E: {pawn, 0, 1, 1, 0},     // CB_PAWN_A_ENC
	0x8F: {rook, 2, 1, 0, 0},     // CB_ROOK_3_ENC
	0x90: {pawn, 3, 1, 1, 0},     // CB_PAWN_D_ENC
	0x91: {bishop, 2, 6, 6, 0},   // CB_BISHOP_3_ENC
	0x92: {queen, 1, 5, 3, 0},    // CB_QUEEN_2_ENC
	0x93: {bishop, 1, 4, 4, 0},   // CB_BISHOP_2_ENC
	0x94: {queen, 1, 0, 2, 0},    // CB_QUEEN_2_ENC
	0x95: {queen, 1, 2, 0, 0},    // CB_QUEEN_2_ENC
	0x96: {queen, 0, 7, 7, 0},    // CB_QUEEN_1_ENC
	0x97: {bishop, 0, 2, 2, 0},   // CB_BISHOP_1_ENC
	0x98: {rook, 1, 5, 0, 0},     // CB_ROOK_2_ENC
	0x99: {queen, 0, 5, 0, 0},    // CB_QUEEN_1_ENC
	0x9A: {rook, 2, 0, 3, 0},     // CB_ROOK_3_ENC
	0x9B: {knight, 2, 2, 1, 0},   // CB_KNIGHT_3_ENC
	0x9C: {rook, 0, 0, 6, 0},     // CB_ROOK_1_ENC
	0x9D: {rook, 2, 0, 5, 0},     // CB_ROOK_3_ENC
	0x9E: {pawn, 5, 0, 2, 0},     // CB_PAWN_F_ENC
	0xA0: {queen, 1, 3, 3, 0},    // CB_QUEEN_2_ENC
	0xA1: {rook, 1, 4, 0, 0},     // CB_ROOK_2_ENC
	0xA2: {bishop, 1, 5, 3, 0},   // CB_BISHOP_2_ENC
	0xA3: {knight, 2, -2, 1, 0},  // CB_KNIGHT_3_ENC
	0xA4: {pawn, 1, -1, 1, 0},    // CB_PAWN_B_ENC
	0xA5: {queen, 0, 0, 1, 0},    // CB_QUEEN_1_ENC
	0xA6: {rook, 1, 1, 0, 0},     // CB_ROOK_2_ENC
	0xA7: {queen, 0, 1, 7, 0},    // CB_QUEEN_1_ENC
	0xA8: {queen, 2, 6, 0, 0},    // CB_QUEEN_3_ENC
	0xA9: {rook, 1, 0, 2, 0},     // CB_ROOK_2_ENC
	0xAB: {bishop, 2, 1, 7, 0},   // CB_BISHOP_3_ENC
	0xAC: {knight, 2, -2, -1, 0}, // CB_KNIGHT_3_ENC
	0xAE: {bishop, 0, 6, 2, 0},   // CB_BISHOP_1_ENC
	0xB0: {queen, 2, 0, 5, 0},    // CB_QUEEN_3_ENC
	0xB1: {king, 0, 7, 7, 0},     // CB_KING_ENC
	0xB2: {king, 0, 7, 0, 0},     // CB_KING_ENC
	0xB3: {bishop, 2, 5, 3, 0},   // CB_BISHOP_3_ENC
	0xB4: {queen, 0, 2, 2, 0},    // CB_QUEEN_1_ENC
	0xB5: {king, 0, -2, 0, -1},   // CB_KING_ENC
	0xB6: {queen, 1, 6, 2, 0},    // CB_QUEEN_2_ENC
	0xB7: {bishop, 0, 2, 6, 0},   // CB_BISHOP_1_ENC
	0xB8: {queen, 0, 0, 2, 0},    // CB_QUEEN_1_ENC
	0xB9: {bishop, 2, 2, 2, 0},   // CB_BISHOP_3_ENC
	0xBA: {knight, 0, -2, -1, 0}, // CB_KNIGHT_1_ENC
	0xBB: {pawn, 6, 0, 1, 0},     // CB_PAWN_G_ENC
	0xBC: {pawn, 6, 1, 1, 0},     // CB_PAWN_G_ENC
	0xBD: {queen, 0, 5, 5, 0},    // CB_QUEEN_1_ENC
	0xBE: {queen, 0, 2, 0, 0},    // CB_QUEEN_1_ENC
	0xBF: {queen, 0, 3, 3, 0},    // CB_QUEEN_1_ENC
	0xC0: {knight, 2, 1, 2, 0},   // CB_KNIGHT_3_ENC
	0xC1: {pawn, 0, 0, 2, 0},     // CB_PAWN_A_ENC
	0xC2: {king, 0, 0, 7, 0},     // CB_KING_ENC
	0xC3: {bishop, 0, 5, 5, 0},   // CB_BISHOP_1_ENC
	0xC4: {knight, 1, 2, 1, 0},   // CB_KNIGHT_2_ENC
	0xC5: {pawn, 3, 0, 1, 0},     // CB_PAWN_D_ENC
	0xC6: {rook, 0, 2, 0, 0},     // CB_ROOK_1_ENC
	0xC8: {bishop, 2, 7, 1, 0},   // CB_BISHOP_3_ENC
	0xC9: {knight, 2, -1, -2, 0}, // CB_KNIGHT_3_ENC
	0xCA: {queen, 1, 3, 0, 0},    // CB_QUEEN_2_ENC
	0xCB: {queen, 0, 0, 3, 0},    // CB_QUEEN_1_ENC
	0xCD: {rook, 2, 2, 0, 0},     // CB_ROOK_3_ENC
	0xCE: {queen, 2, 5, 3, 0},    // CB_QUEEN_3_ENC
	0xD1: {queen, 2, 0, 6, 0},    // CB_QUEEN_3_ENC
	0xD2: {queen, 0, 6, 0, 0},    // CB_QUEEN_1_ENC
	0xD3: {queen, 1, 4, 0, 0},    // CB_QUEEN_2_ENC
	0xD4: {knight, 0, -1, -2, 0}, // CB_KNIGHT_1_ENC
	0xD6: {rook, 2, 7, 0, 0},     // CB_ROOK_3_ENC
	0xD7: {rook, 0, 0, 4, 0},     // CB_ROOK_1_ENC
	0xD8: {king, 0, 1, 0, 0},     // CB_KING_ENC
	0xD9: {bishop, 0, 4, 4, 0},   // CB_BISHOP_1_ENC
	0xDA: {pawn, 2, 0, 2, 0},     // CB_PAWN_C_ENC
	0xDB: {queen, 2, 7, 1, 0},    // CB_QUEEN_3_ENC
	0xDD: {knight, 0, 1, -2, 0},  // CB_KNIGHT_1_ENC
	0xDE: {pawn, 5, -1, 1, 0},    // CB_PAWN_F_ENC
	0xDF: {pawn, 6, 0, 2, 0},     // CB_PAWN_G_ENC
	0xE0: {pawn, 2, 1, 1, 0},     // CB_PAWN_C_ENC
	0xE1: {bishop, 0, 3, 3, 0},   // CB_BISHOP_1_ENC
	0xE2: {rook, 1, 0, 7, 0},     // CB_ROOK_2_ENC
	0xE3: {knight, 2, -1, 2, 0},  // CB_KNIGHT_3_ENC
	0xE4: {bishop, 0, 7, 7, 0},   // CB_BISHOP_1_ENC
	0xE5: {queen, 1, 0, 1, 0},    // CB_QUEEN_2_ENC
	0xE6: {rook, 0, 0, 7, 0},     // CB_ROOK_1_ENC
	0xE7: {queen, 2, 1, 1, 0},    // CB_QUEEN_3_ENC
	0xE8: {queen, 2, 6, 6, 0},    // CB_QUEEN_3_ENC
	0xE9: {knight, 0, -2, 1, 0},  // CB_KNIGHT_1_ENC
	0xEA: {queen, 1, 0, 5, 0},    // CB_QUEEN_2_ENC
	0xEB: {queen, 0, 3, 0, 0},    // CB_QUEEN_1_ENC
	0xEC: {knight, 2, 1, -2, 0},  // CB_KNIGHT_3_ENC
	0xED: {rook, 2, 3, 0, 0},     // CB_ROOK_3_ENC
	0xEE: {rook, 1, 0, 4, 0},     // CB_ROOK_2_ENC
	0xEF: {queen, 1, 7, 0, 0},    // CB_QUEEN_2_ENC
	0xF0: {queen, 2, 1, 0, 0},    // CB_QUEEN_3_ENC
	0xF1: {queen, 2, 3, 5, 0},    // CB_QUEEN_3_ENC
	0xF2: {bishop, 1, 2, 6, 0},   // CB_BISHOP_2_ENC
	0xF3: {bishop, 1, 6, 2, 0},   // CB_BISHOP_2_ENC
	0xF4: {queen, 2, 5, 0, 0},    // CB_QUEEN_3_ENC
	0xF5: {pawn, 0, -1, 1, 0},    // CB_PAWN_A_ENC
	0xF6: {bishop, 1, 1, 1, 0},   // CB_BISHOP_2_ENC
	0xF8: {rook, 0, 0, 2, 0},     // CB_ROOK_1_ENC
	0xF9: {pawn, 3, -1, 1, 0},    // CB_PAWN_D_ENC
	0xFA: {knight, 0, -1, 2, 0},  // CB_KNIGHT_1_ENC
	0xFB: {rook, 1, 0, 5, 0},     // CB_ROOK_2_ENC
	0xFC: {bishop, 2, 6, 2, 0},   // CB_BISHOP_3_ENC
	0xFD: {bishop, 2, 7, 7, 0},   // CB_BISHOP_3_ENC
	0xFE: {knight, 1, -1, 2, 0},  // CB_KNIGHT_2_ENC
	0xFF: {pawn, 4, 0, 2, 0},     // CB_PAWN_E_ENC
}

var legacyDeob2 = [256]byte{
	0xA2, 0x95, 0x43, 0xF5, 0xC1, 0x3D, 0x4A, 0x6C, 0x53, 0x83, 0xCC, 0x7C, 0xFF, 0xAE, 0x68, 0xAD,
	0xD1, 0x92, 0x8B, 0x8D, 0x35, 0x81, 0x5E, 0x74, 0x26, 0x8E, 0xAB, 0xCA, 0xFD, 0x9A, 0xF3, 0xA0,
	0xA5, 0x15, 0xFC, 0xB1, 0x1E, 0xED, 0x30, 0xEA, 0x22, 0xEB, 0xA7, 0xCD, 0x4E, 0x6F, 0x2E, 0x24,
	0x32, 0x94, 0x41, 0x8C, 0x6E, 0x58, 0x82, 0x50, 0xBB, 0x02, 0x8A, 0xD8, 0xFA, 0x60, 0xDE, 0x52,
	0xBA, 0x46, 0xAC, 0x29, 0x9D, 0xD7, 0xDF, 0x08, 0x21, 0x01, 0x66, 0xA3, 0xF1, 0x19, 0x27, 0xB5,
	0x91, 0xD5, 0x42, 0x0E, 0xB4, 0x4C, 0xD9, 0x18, 0x5F, 0xBC, 0x25, 0xA6, 0x96, 0x04, 0x56, 0x6A,
	0xAA, 0x33, 0x1C, 0x2B, 0x73, 0xF0, 0xDD, 0xA4, 0x37, 0xD3, 0xC5, 0x10, 0xBF, 0x5A, 0x23, 0x34,
	0x75, 0x5B, 0xB8, 0x55, 0xD2, 0x6B, 0x09, 0x3A, 0x57, 0x12, 0xB3, 0x77, 0x48, 0x85, 0x9B, 0x0F,
	0x9E, 0xC7, 0xC8, 0xA1, 0x7F, 0x7A, 0xC0, 0xBD, 0x31, 0x6D, 0xF6, 0x3E, 0xC3, 0x11, 0x71, 0xCE,
	0x7D, 0xDA, 0xA8, 0x54, 0x90, 0x97, 0x1F, 0x44, 0x40, 0x16, 0xC9, 0xE3, 0x2C, 0xCB, 0x84, 0xEC,
	0x9F, 0x3F, 0x5C, 0xE6, 0x76, 0x0B, 0x3C, 0x20, 0xB7, 0x36, 0x00, 0xDC, 0xE7, 0xF9, 0x4F, 0xF7,
	0xAF, 0x06, 0x07, 0xE0, 0x1A, 0x0A, 0xA9, 0x4B, 0x0C, 0xD6, 0x63, 0x87, 0x89, 0x1D, 0x13, 0x1B,
	0xE4, 0x70, 0x05, 0x47, 0x67, 0x7B, 0x2F, 0xEE, 0xE2, 0xE8, 0x98, 0x0D, 0xEF, 0xCF, 0xC4, 0xF4,
	0xFB, 0xB0, 0x17, 0x99, 0x64, 0xF2, 0xD4, 0x2A, 0x03, 0x4D, 0x78, 0xC6, 0xFE, 0x65, 0x86, 0x88,
	0x79, 0x45, 0x3B, 0xE5, 0x49, 0x8F, 0x2D, 0xB9, 0xBE, 0x62, 0x93, 0x14, 0xE9, 0xD0, 0x38, 0x9C,
	0xB2, 0xC2, 0x59, 0x5D, 0xB6, 0x72, 0x51, 0xF8, 0x28, 0x7E, 0x61, 0x39, 0xE1, 0xDB, 0x69, 0x80,
}

type legacyCoord struct {
	x, y int
	ok   bool
}
type legacyCell struct{ typ, no int }
type legacyRefState struct {
	pos  [8][8]legacyCell // [file][rank], as in cbh2pgn
	list [13][8]legacyCoord
}

func newLegacyRefState() legacyRefState {
	var s legacyRefState
	put := func(x, y, typ, no int) { s.pos[x][y] = legacyCell{typ, no}; s.list[typ][no] = legacyCoord{x, y, true} }
	put(0, 0, cbWRook, 0)
	put(1, 0, cbWKnight, 0)
	put(2, 0, cbWBishop, 0)
	put(3, 0, cbWQueen, 0)
	put(4, 0, cbWKing, 0)
	put(5, 0, cbWBishop, 1)
	put(6, 0, cbWKnight, 1)
	put(7, 0, cbWRook, 1)
	put(0, 7, cbBRook, 0)
	put(1, 7, cbBKnight, 0)
	put(2, 7, cbBBishop, 0)
	put(3, 7, cbBQueen, 0)
	put(4, 7, cbBKing, 0)
	put(5, 7, cbBBishop, 1)
	put(6, 7, cbBKnight, 1)
	put(7, 7, cbBRook, 1)
	for x := 0; x < 8; x++ {
		put(x, 1, cbWPawn, x)
		put(x, 6, cbBPawn, x)
	}
	return s
}

func cbTypeFor(kind, side int) int {
	if side == white {
		switch kind {
		case king:
			return cbWKing
		case queen:
			return cbWQueen
		case rook:
			return cbWRook
		case bishop:
			return cbWBishop
		case knight:
			return cbWKnight
		case pawn:
			return cbWPawn
		}
	} else {
		switch kind {
		case king:
			return cbBKing
		case queen:
			return cbBQueen
		case rook:
			return cbBRook
		case bishop:
			return cbBBishop
		case knight:
			return cbBKnight
		case pawn:
			return cbBPawn
		}
	}
	return 0
}
func cbTypeToBoard(t int) int {
	switch t {
	case cbWKing:
		return king
	case cbWQueen:
		return queen
	case cbWRook:
		return rook
	case cbWBishop:
		return bishop
	case cbWKnight:
		return knight
	case cbWPawn:
		return pawn
	case cbBKing:
		return -king
	case cbBQueen:
		return -queen
	case cbBRook:
		return -rook
	case cbBBishop:
		return -bishop
	case cbBKnight:
		return -knight
	case cbBPawn:
		return -pawn
	}
	return 0
}
func cbIsKing(t int) bool { return t == cbWKing || t == cbBKing }
func cbIsPawn(t int) bool { return t == cbWPawn || t == cbBPawn }
func mod8(x int) int {
	x %= 8
	if x < 0 {
		x += 8
	}
	return x
}

func (s *legacyRefState) decreasePieceNr(typ, n int) {
	if typ <= 0 || typ >= 13 || n < 0 || n >= 8 {
		return
	}
	// Exact cbh2pgn semantics: every later numbered non-pawn/non-king piece shifts down.
	for i := n; i < 7; i++ {
		s.list[typ][i] = s.list[typ][i+1]
	}
	s.list[typ][7] = legacyCoord{}
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			c := s.pos[x][y]
			if c.typ == typ && c.no > n {
				c.no--
				s.pos[x][y] = c
			}
		}
	}
}

func (s *legacyRefState) oneByteDestination(side int, sp legacyTokenSpec) (legacyCoord, legacyCoord, int, error) {
	typ := cbTypeFor(sp.kind, side)
	if typ == 0 || sp.no < 0 || sp.no >= 8 || !s.list[typ][sp.no].ok {
		return legacyCoord{}, legacyCoord{}, typ, fmt.Errorf("stuk %d/%d bestaat niet", sp.kind, sp.no)
	}
	from := s.list[typ][sp.no]
	dx, dy := sp.dx, sp.dy
	if typ == cbBPawn {
		dx = -dx
		dy = -dy
	}
	to := legacyCoord{mod8(from.x + dx), mod8(from.y + dy), true}
	return from, to, typ, nil
}

func (s *legacyRefState) applyOneByte(side int, tok byte, sp legacyTokenSpec, from, to legacyCoord, typ int) {
	s.pos[from.x][from.y] = legacyCell{}
	tgt := s.pos[to.x][to.y]
	if tgt.typ != 0 && !cbIsKing(tgt.typ) && !cbIsPawn(tgt.typ) {
		s.decreasePieceNr(tgt.typ, tgt.no)
	}
	s.pos[to.x][to.y] = legacyCell{typ, sp.no}
	s.list[typ][sp.no] = to
	// Exact reference castle bookkeeping.
	if typ == cbWKing && tok == 0x76 {
		s.moveRookForCastle(cbWRook, 7, 0, 5, 0)
	}
	if typ == cbBKing && tok == 0x76 {
		s.moveRookForCastle(cbBRook, 7, 7, 5, 7)
	}
	if typ == cbWKing && tok == 0xB5 {
		s.moveRookForCastle(cbWRook, 0, 0, 3, 0)
	}
	if typ == cbBKing && tok == 0xB5 {
		s.moveRookForCastle(cbBRook, 0, 7, 3, 7)
	}
}
func (s *legacyRefState) moveRookForCastle(typ, fx, fy, tx, ty int) {
	s.pos[fx][fy] = legacyCell{}
	for i := 0; i < 8; i++ {
		c := s.list[typ][i]
		if c.ok && c.x == fx && c.y == fy {
			s.list[typ][i] = legacyCoord{tx, ty, true}
			s.pos[tx][ty] = legacyCell{typ, i}
			return
		}
	}
}

func absCBToXY(v int) (int, int) { return v / 8, v % 8 }

func (s *legacyRefState) applyTwoByte(side int, from, to legacyCoord, promCode int) (promo int, error error) {
	cell := s.pos[from.x][from.y]
	if cell.typ == 0 {
		return 0, errors.New("2-byte bronveld is leeg")
	}
	s.pos[from.x][from.y] = legacyCell{}
	tgt := s.pos[to.x][to.y]
	if tgt.typ != 0 && !cbIsKing(tgt.typ) && !cbIsPawn(tgt.typ) {
		s.decreasePieceNr(tgt.typ, tgt.no)
	}
	if !cbIsPawn(cell.typ) {
		s.pos[to.x][to.y] = cell
		s.list[cell.typ][cell.no] = to
		return 0, nil
	}
	promotedType := 0
	if cell.typ == cbWPawn && to.y == 7 || cell.typ == cbBPawn && to.y == 0 {
		switch promCode {
		case 0:
			promo = queen
		case 1:
			promo = rook
		case 2:
			promo = bishop
		case 3:
			promo = knight
		}
		promotedType = cbTypeFor(promo, side)
	}
	if promotedType != 0 {
		free := -1
		for i := 0; i < 8; i++ {
			if !s.list[promotedType][i].ok {
				free = i
				break
			}
		}
		if free < 0 {
			return 0, errors.New("geen vrij promoted-piece slot")
		}
		s.list[promotedType][free] = to
		s.pos[to.x][to.y] = legacyCell{promotedType, free}
		// The original converter intentionally leaves the old pawn-list slot stale.
		return promo, nil
	}
	// Reference assumes a 2-byte pawn move is normally a promotion. Preserve a
	// non-promotion pawn defensively so our legal board does not diverge.
	s.pos[to.x][to.y] = cell
	s.list[cell.typ][cell.no] = to
	return 0, nil
}

func boardFromLegacyState(s legacyRefState, side, castle, ep int) Board {
	var b Board
	b.Side = side
	b.Castle = castle
	b.EP = ep
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			b.Sq[y*8+x] = cbTypeToBoard(s.pos[x][y].typ)
		}
	}
	return b
}

type legacyDecoded struct {
	sans       []string
	variations int64
	fen        string
	startSide  int
	fullmove   int
}

type legacyBranch struct {
	board Board
	state legacyRefState
}

func isLegacySpecial(t byte) bool        { return t == 0x29 || t == 0xDC || t == 0x0C || t == 0x9F }
func subtractToken(raw byte, n int) byte { return byte((int(raw) - n) & 255) }

func decodeLegacyReference(stream []byte, b Board, st legacyRefState, startSide, fullmove int) (legacyDecoded, error) {
	out := legacyDecoded{startSide: startSide, fullmove: fullmove}
	processed := 0
	idx := 0
	stack := make([]legacyBranch, 0, 4)
	for idx < len(stream) {
		tok := subtractToken(stream[idx], processed)
		if !isLegacySpecial(tok) {
			processed = (processed + 1) & 255
		}
		if tok == 0x9F {
			idx++
			continue
		}
		if tok == 0xAA {
			if len(stack) == 0 {
				out.sans = append(out.sans, "--")
			}
			b.Side = -b.Side
			b.EP = -1
			idx++
			continue
		}
		if tok == 0x29 {
			if idx+2 >= len(stream) {
				return out, errors.New("afgebroken 2-byte CBH-zet")
			}
			a := legacyDeob2[subtractToken(stream[idx+1], processed)]
			z := legacyDeob2[subtractToken(stream[idx+2], processed)]
			word := binary.BigEndian.Uint16([]byte{a, z})
			src := int(word & 0x3f)
			dst := int((word >> 6) & 0x3f)
			pc := int((word >> 12) & 3)
			fx, fy := absCBToXY(src)
			tx, ty := absCBToXY(dst)
			from := legacyCoord{fx, fy, true}
			to := legacyCoord{tx, ty, true}
			cell := st.pos[fx][fy]
			promo := 0
			if cbIsPawn(cell.typ) && (ty == 0 || ty == 7) {
				switch pc {
				case 0:
					promo = queen
				case 1:
					promo = rook
				case 2:
					promo = bishop
				case 3:
					promo = knight
				}
			}
			frSq := fy*8 + fx
			toSq := ty*8 + tx
			m, san, e := matchLegal(b, frSq, toSq, promo)
			if e != nil {
				return out, fmt.Errorf("2-byte %s%s: %w", squareName(frSq), squareName(toSq), e)
			}
			gotPromo, e := st.applyTwoByte(b.Side, from, to, pc)
			if e != nil {
				return out, e
			}
			_ = gotPromo
			if len(stack) == 0 {
				out.sans = append(out.sans, san)
			}
			b = b.afterUnchecked(m)
			processed = (processed + 1) & 255
			idx += 3
			continue
		}
		if tok == 0xDC {
			stack = append(stack, legacyBranch{b, st})
			out.variations++
			idx++
			continue
		}
		if tok == 0x0C {
			if idx < len(stream)-1 {
				if len(stack) == 0 {
					return out, errors.New("CBH variatie-pop zonder push")
				}
				q := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				b = q.board
				st = q.state
			}
			idx++
			continue
		}
		sp, ok := legacyOneByte[tok]
		if !ok {
			// Match cbh2pgn 0.1: unrecognised non-special tokens are consumed but
			// do not manufacture a chess move.
			idx++
			continue
		}
		from, to, typ, e := st.oneByteDestination(b.Side, sp)
		if e != nil {
			return out, fmt.Errorf("token 0x%02X: %w", tok, e)
		}
		frSq := from.y*8 + from.x
		toSq := to.y*8 + to.x
		m, san, e := matchLegal(b, frSq, toSq, 0)
		if e != nil {
			return out, fmt.Errorf("token 0x%02X %s%s: %w", tok, squareName(frSq), squareName(toSq), e)
		}
		st.applyOneByte(b.Side, tok, sp, from, to, typ)
		if len(stack) == 0 {
			out.sans = append(out.sans, san)
		}
		b = b.afterUnchecked(m)
		idx++
	}
	return out, nil
}

// Classic-CBH game header bits, copied from cbh2pgn 0.1 get_info_gamelen().
type legacyGameInfo struct {
	customStart, notEncoded, is960, special bool
	length                                  int
}

func parseLegacyGameInfo(h []byte) (legacyGameInfo, error) {
	if len(h) < 4 {
		return legacyGameInfo{}, ioErrShort("CBG-header")
	}
	x := binary.BigEndian.Uint32(h[:4])
	return legacyGameInfo{
		customStart: (x & 0x40000000) != 0,
		notEncoded:  (x & 0x80000000) != 0,
		special:     (x & 0x04000000) != 0,
		is960:       (x & 0x0A000000) != 0,
		length:      int(x & 0x00FFFFFF),
	}, nil
}
func ioErrShort(what string) error { return fmt.Errorf("%s te kort", what) }

func decodeLegacyCustomStart(data []byte) (legacyRefState, Board, string, int, int, error) {
	var st legacyRefState
	if len(data) < 28 {
		return st, Board{}, "", white, 1, errors.New("CBH beginstelling-header te kort")
	}
	epFile := int(data[1] & 0x07)
	blackTurn := (data[1] & 0x10) != 0
	cr := data[2]
	full := int(data[3])
	if full <= 0 {
		full = 1
	}
	bits := data[4:28]
	bitpos := 0
	sq := 0
	put := func(x, y, typ int) error {
		no := 0
		if typ == cbWKing || typ == cbBKing {
			no = 0
		} else {
			for no < 8 && st.list[typ][no].ok {
				no++
			}
			if no >= 8 {
				return errors.New("te veel stukken van één type in beginstelling")
			}
		}
		st.pos[x][y] = legacyCell{typ, no}
		st.list[typ][no] = legacyCoord{x, y, true}
		return nil
	}
	readBit := func() int {
		if bitpos >= len(bits)*8 {
			return -1
		}
		v := int((bits[bitpos/8] >> (7 - uint(bitpos%8))) & 1)
		bitpos++
		return v
	}
	for sq < 64 && bitpos < len(bits)*8 {
		first := readBit()
		if first < 0 {
			break
		}
		if first == 0 {
			sq++
			continue
		}
		code := 1
		for k := 0; k < 4; k++ {
			z := readBit()
			if z < 0 {
				return st, Board{}, "", white, 1, errors.New("afgebroken stukcode in beginstelling")
			}
			code = (code << 1) | z
		}
		typ := 0
		switch code {
		case 0b10001:
			typ = cbWKing
		case 0b10010:
			typ = cbWQueen
		case 0b10011:
			typ = cbWKnight
		case 0b10100:
			typ = cbWBishop
		case 0b10101:
			typ = cbWRook
		case 0b10110:
			typ = cbWPawn
		case 0b11001:
			typ = cbBKing
		case 0b11010:
			typ = cbBQueen
		case 0b11011:
			typ = cbBKnight
		case 0b11100:
			typ = cbBBishop
		case 0b11101:
			typ = cbBRook
		case 0b11110:
			typ = cbBPawn
		default:
			return st, Board{}, "", white, 1, fmt.Errorf("onbekende beginstelling-stukcode %05b", code)
		}
		x, y := absCBToXY(sq)
		if e := put(x, y, typ); e != nil {
			return st, Board{}, "", white, 1, e
		}
		sq++
	}
	side := white
	if blackTurn {
		side = black
	}
	castle := 0
	if cr&2 != 0 {
		castle |= wk
	}
	if cr&1 != 0 {
		castle |= wq
	}
	if cr&8 != 0 {
		castle |= bk
	}
	if cr&4 != 0 {
		castle |= bq
	}
	ep := -1
	if epFile > 0 {
		f := epFile - 1
		r := 5
		if blackTurn {
			r = 2
		}
		if f >= 0 && f < 8 {
			ep = r*8 + f
		}
	}
	b := boardFromLegacyState(st, side, castle, ep)
	fen := legacyFEN(b, full)
	return st, b, fen, side, full, nil
}

func legacyFEN(b Board, full int) string {
	s := ""
	for r := 7; r >= 0; r-- {
		emptyN := 0
		for f := 0; f < 8; f++ {
			p := b.Sq[r*8+f]
			if p == 0 {
				emptyN++
				continue
			}
			if emptyN > 0 {
				s += fmt.Sprint(emptyN)
				emptyN = 0
			}
			ch := "?"
			switch abs(p) {
			case king:
				ch = "k"
			case queen:
				ch = "q"
			case rook:
				ch = "r"
			case bishop:
				ch = "b"
			case knight:
				ch = "n"
			case pawn:
				ch = "p"
			}
			if p > 0 {
				ch = string(ch[0] - 32)
			}
			s += ch
		}
		if emptyN > 0 {
			s += fmt.Sprint(emptyN)
		}
		if r > 0 {
			s += "/"
		}
	}
	if b.Side == white {
		s += " w "
	} else {
		s += " b "
	}
	c := ""
	if b.Castle&wk != 0 {
		c += "K"
	}
	if b.Castle&wq != 0 {
		c += "Q"
	}
	if b.Castle&bk != 0 {
		c += "k"
	}
	if b.Castle&bq != 0 {
		c += "q"
	}
	if c == "" {
		c = "-"
	}
	s += c + " "
	if b.EP >= 0 {
		s += squareName(b.EP)
	} else {
		s += "-"
	}
	if full <= 0 {
		full = 1
	}
	return fmt.Sprintf("%s 0 %d", s, full)
}
