module Components.NodeMessageService exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.send
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        service =
                            Point.getText o.node.points Point.typeService ""

                        isTwilio =
                            service == Point.valueTwilio

                        isSMTP =
                            service == Point.valueSMTP

                        isNtfy =
                            service == Point.valueNtfy
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , optionInput Point.typeService
                        "Service"
                        [ ( Point.valueTwilio, "Twilio SMS" )
                        , ( Point.valueSMTP, "Email (SMTP)" )
                        , ( Point.valueNtfy, "ntfy push" )
                        ]
                    , viewIf isTwilio <|
                        textInput Point.typeSID "SID" ""
                    , viewIf isTwilio <|
                        textInput Point.typeAuthToken "Auth Token" ""
                    , viewIf isTwilio <|
                        textInput Point.typeFrom "From" "+15555555555"
                    , viewIf isSMTP <|
                        textInput Point.typeURL "Server" "smtp.example.com:587"
                    , viewIf isSMTP <|
                        textInput Point.typeUsername "Username" ""
                    , viewIf isSMTP <|
                        textInput Point.typeAuthToken "Password" ""
                    , viewIf isSMTP <|
                        textInput Point.typeFrom "From" "siot@example.com"
                    , viewIf isNtfy <|
                        textInput Point.typeURL "Server" "https://ntfy.sh"
                    , viewIf isNtfy <|
                        textInput Point.typeTopic "Topic" ""
                    , viewIf isNtfy <|
                        textInput Point.typeAuthToken "Access Token" "(optional)"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
