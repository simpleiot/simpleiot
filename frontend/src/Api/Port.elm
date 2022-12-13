port module Api.Port exposing
    ( encodeClipboard
    , out
    )

import Json.Encode


port out : Json.Encode.Value -> Cmd msg


encodeClipboard : String -> Json.Encode.Value
encodeClipboard s =
    Json.Encode.object
        [ ( "action", Json.Encode.string "CLIPBOARD" )
        , ( "data", Json.Encode.string s )
        ]
